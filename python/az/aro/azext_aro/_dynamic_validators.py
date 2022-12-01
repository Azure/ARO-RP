# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

import ipaddress
import re
from itertools import tee

from azure.cli.core.commands.client_factory import get_mgmt_service_client
from azure.cli.core.profiles import ResourceType
from azure.cli.core.azclierror import CLIInternalError, InvalidArgumentValueError, \
    RequiredArgumentMissingError
from azure.core.exceptions import ResourceNotFoundError
from knack.log import get_logger
from msrestazure.tools import is_valid_resource_id
from msrestazure.tools import parse_resource_id
from msrestazure.azure_exceptions import CloudError
from azext_aro._validators import validate_vnet, validate_cidr
from azext_aro._rbac import has_role_assignment_on_resource
from humanfriendly.terminal.spinners import AutomaticSpinner


logger = get_logger(__name__)


def can_do_action(perms, action):
    for perm in perms:
        for perm_action in perm.actions:
            clean = re.escape(perm_action)
            clean = re.match("(?i)^" + clean.replace(r"\*", ".*") + "$", action)
            if clean:
                return None
        for not_action in perm.not_actions:
            clean = re.escape(not_action)
            clean = re.match("(?i)^" + clean.replace(r"\*", ".*") + "$", action)
            if clean:
                return f"{action} permission is disabled"
    return f"{action} permission is missing"


def validate_resource(client, key, resource, actions):
    perms = client.permissions.list_for_resource(resource['resource_group'],
                                                 resource['namespace'],
                                                 "",
                                                 resource['type'],
                                                 resource['name'])

    errors = []
    for action in actions:
        perms, perms_copy = tee(perms)
        perms_list = list(perms_copy)
        error = can_do_action(perms_list, action)
        if error is not None:
            row = [key, resource['name'], error]
            errors.append(row)

    return errors


def get_subnet(client, subnet, subnet_parts):
    try:
        subnet_obj = client.subnets.get(subnet_parts['resource_group'],
                                        subnet_parts['name'],
                                        subnet_parts['child_name_1'])
    except Exception as err:
        if isinstance(err, ResourceNotFoundError):
            raise InvalidArgumentValueError(
                f"Invalid -- subnet, error when getting '{subnet}': {str(err)}") from err
        raise CLIInternalError(
            f"Unexpected error when getting subnet '{subnet}': {str(err)}") from err

    return subnet_obj


def dyn_validate_vnet(key):
    def _validate_vnet(cmd, namespace):
        errors = []
        with AutomaticSpinner("Validating Virtual Network Permissions"):
            vnet = getattr(namespace, key)

            if not is_valid_resource_id(vnet):
                raise RequiredArgumentMissingError(
                    f"Must specify --vnet if --{key.replace('_', '-')} is not an id.")

            validate_vnet(cmd, namespace)

            parts = parse_resource_id(vnet)

            network_client = get_mgmt_service_client(
                cmd.cli_ctx, ResourceType.MGMT_NETWORK)

            auth_client = get_mgmt_service_client(
                cmd.cli_ctx, ResourceType.MGMT_AUTHORIZATION, api_version="2015-07-01")

            try:
                network_client.virtual_networks.get(parts['resource_group'], parts['name'])
            except Exception as err:
                if isinstance(err, ResourceNotFoundError):
                    raise InvalidArgumentValueError(
                        f"Invalid --{key.replace('_', '-')}, error when getting '{vnet}': {str(err)}") from err
                raise CLIInternalError(
                    f"Unexpected error when getting subnet '{vnet}': {str(err)}") from err

            errors = validate_resource(auth_client, key, parts, [
                "Microsoft.Network/virtualNetworks/join/action",
                "Microsoft.Network/virtualNetworks/read",
                "Microsoft.Network/virtualNetworks/write",
                "Microsoft.Network/virtualNetworks/subnets/join/action",
                "Microsoft.Network/virtualNetworks/subnets/read",
                "Microsoft.Network/virtualNetworks/subnets/write", ])

        return errors

    return _validate_vnet


def dyn_validate_subnet(key):
    def _validate_subnet(cmd, namespace):
        errors = []
        with AutomaticSpinner(f"Validating {key} permissions"):
            subnet = getattr(namespace, key)

            if not is_valid_resource_id(subnet):
                if not namespace.vnet:
                    raise RequiredArgumentMissingError(
                        f"Must specify --vnet if --{key.replace('_', '-')} is not an id.")

                validate_vnet(cmd, namespace)

                subnet = namespace.vnet + '/subnets/' + subnet
                setattr(namespace, key, subnet)

            parts = parse_resource_id(subnet)

            network_client = get_mgmt_service_client(
                cmd.cli_ctx, ResourceType.MGMT_NETWORK)

            auth_client = get_mgmt_service_client(
                cmd.cli_ctx, ResourceType.MGMT_AUTHORIZATION, api_version="2015-07-01")

            try:
                subnet_obj = network_client.subnets.get(parts['resource_group'],
                                                        parts['name'],
                                                        parts['child_name_1'])
                route_table_obj = subnet_obj.route_table
                if route_table_obj is None:
                    raise ResourceNotFoundError("Subnet is missing route table")
            except Exception as err:
                if isinstance(err, ResourceNotFoundError):
                    raise InvalidArgumentValueError(
                        f"Invalid --{key.replace('_', '-')}, error when getting '{subnet}': {str(err)}") from err
                raise CLIInternalError(
                    f"Unexpected error when getting subnet '{subnet}': {str(err)}") from err

            route_parts = parse_resource_id(route_table_obj.id)

            errors = validate_resource(auth_client, f"{key}_route_table", route_parts, [
                "Microsoft.Network/routeTables/join/action",
                "Microsoft.Network/routeTables/read",
                "Microsoft.Network/routeTables/write", ])

        return errors

    return _validate_subnet


def dyn_validate_cidr_ranges():
    def _validate_cidr_ranges(cmd, namespace):
        addresses = []
        with AutomaticSpinner("Validating no overlapping CIDR Ranges on subnets"):
            ERROR_KEY = "CIDR Range"
            master_subnet = namespace.master_subnet
            worker_subnet = namespace.worker_subnet
            pod_cidr = namespace.pod_cidr
            service_cidr = namespace.service_cidr

            worker_parts = parse_resource_id(worker_subnet)
            master_parts = parse_resource_id(master_subnet)

            fn = validate_cidr("pod_cidr")
            fn(namespace)
            fn = validate_cidr("service_cidr")
            fn(namespace)

            cidr_array = {}

            if pod_cidr is not None:
                cidr_array["Pod CIDR"] = ipaddress.IPv4Network(pod_cidr)
            if service_cidr is not None:
                cidr_array["Service CIDR"] = ipaddress.IPv4Network(service_cidr)

            network_client = get_mgmt_service_client(
                cmd.cli_ctx, ResourceType.MGMT_NETWORK)

            worker_subnet_obj = get_subnet(network_client, worker_subnet, worker_parts)

            if worker_subnet_obj.address_prefix is None:
                for address in worker_subnet_obj.address_prefixes:
                    cidr_array["Worker Subnet CIDR -- " + address] = ipaddress.IPv4Network(address)
            else:
                cidr_array["Worker Subnet CIDR"] = ipaddress.IPv4Network(worker_subnet_obj.address_prefix)

            master_subnet_obj = get_subnet(network_client, master_subnet, master_parts)

            if master_subnet_obj.address_prefix is None:
                for address in master_subnet_obj.address_prefixes:
                    cidr_array["Master Subnet CIDR -- " + address] = ipaddress.IPv4Network(address)
            else:
                cidr_array["Master Subnet CIDR"] = ipaddress.IPv4Network(master_subnet_obj.address_prefix)

            ipv4_zero = ipaddress.IPv4Network("0.0.0.0/0")

            for item in cidr_array.items():
                key = item[0]
                cidr = item[1]
                if not cidr.overlaps(ipv4_zero):
                    addresses.append([ERROR_KEY, key, f"{cidr} is not valid as it does not overlap with {ipv4_zero}"])
                for item2 in cidr_array.items():
                    compare = item2[1]
                    if cidr is not compare:
                        if cidr.overlaps(compare):
                            addresses.append([ERROR_KEY, key, f"{cidr} is not valid as it overlaps with {compare}"])

        return addresses

    return _validate_cidr_ranges


def dyn_validate_resource_permissions(service_principle_ids, resources):
    def _validate_resource_permissions(cmd,
                                       namespace  # pylint: disable=unused-argument
                                       ):
        errors = []
        with AutomaticSpinner("Validating resource permissions"):
            for sp_id in service_principle_ids:
                for role in sorted(resources):
                    for resource in resources[role]:
                        try:
                            resource_contributor_exists = has_role_assignment_on_resource(cmd.cli_ctx,
                                                                                          resource,
                                                                                          sp_id,
                                                                                          role)
                            if not resource_contributor_exists:
                                parts = parse_resource_id(resource)
                                errors.append(["Resource Permissions",
                                               parts['type'],
                                               f"Resource {parts['name']} is missing role assignment {role}"])
                        except CloudError as e:
                            logger.error(e.message)
                            raise
        return errors
    return _validate_resource_permissions


def validate_cluster_create(cmd,  # pylint: disable=unused-argument
                            client,  # pylint: disable=unused-argument
                            resource_group_name,  # pylint: disable=unused-argument
                            master_subnet,  # pylint: disable=unused-argument
                            worker_subnet,  # pylint: disable=unused-argument
                            vnet,  # pylint: disable=unused-argument
                            pod_cidr,  # pylint: disable=unused-argument
                            service_cidr,  # pylint: disable=unused-argument
                            location,  # pylint: disable=unused-argument
                            client_id,  # pylint: disable=unused-argument
                            master_vm_size,  # pylint: disable=unused-argument
                            disk_encrpytion_set_id,  # pylint: disable=unused-argument
                            worker_vm_size,  # pylint: disable=unused-argument
                            worker_vm_disk_size_gb,  # pylint: disable=unused-argument
                            worker_count,  # pylint: disable=unused-argument
                            resources,  # pylint: disable=unused-argument
                            service_principle_ids):  # pylint: disable=unused-argument
    error_object = []

    error_object.append(dyn_validate_vnet("vnet"))
    error_object.append(dyn_validate_subnet("master_subnet"))
    error_object.append(dyn_validate_subnet("worker_subnet"))
    error_object.append(dyn_validate_cidr_ranges())
    error_object.append(dyn_validate_resource_permissions(service_principle_ids, resources))

    return error_object
