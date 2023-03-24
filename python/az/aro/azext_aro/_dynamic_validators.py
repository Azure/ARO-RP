# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

import ipaddress
import re
from itertools import tee

from azure.cli.core.commands.client_factory import get_mgmt_service_client
from azure.cli.core.commands.validators import get_default_location_from_resource_group
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
import azext_aro.custom


logger = get_logger(__name__)


def can_do_action(perms, action):
    for perm in perms:
        for not_action in perm.not_actions:
            match = re.escape(not_action)
            match = re.match("(?i)^" + match.replace(r"\*", ".*") + "$", action)
            if match:
                return f"{action} permission is disabled"
        for perm_action in perm.actions:
            match = re.escape(perm_action)
            match = re.match("(?i)^" + match.replace(r"\*", ".*") + "$", action)
            if match:
                return None

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
    except ResourceNotFoundError as err:
        raise InvalidArgumentValueError(
            f"Invalid -- subnet, error when getting '{subnet}': {str(err)}") from err

    except Exception as err:
        raise CLIInternalError(
            f"Unexpected error when getting subnet '{subnet}': {str(err)}") from err

    return subnet_obj


def get_clients(key, cmd):
    parts = parse_resource_id(key)
    network_client = get_mgmt_service_client(
        cmd.cli_ctx, ResourceType.MGMT_NETWORK)

    auth_client = get_mgmt_service_client(
        cmd.cli_ctx, ResourceType.MGMT_AUTHORIZATION, api_version="2015-07-01")

    return parts, network_client, auth_client


# Function to create a progress tracker decorator for the dynamic validation functions
def get_progress_tracker(msg):
    def progress_tracking(func):
        def inner(cmd, namespace):
            hook = cmd.cli_ctx.get_progress_controller()
            hook.add(message=msg)

            errors = func(cmd, namespace)

            hook.end()

            return errors
        return inner
    return progress_tracking


# Validating that the virtual network has the correct permissions
def dyn_validate_vnet(key):
    prog = get_progress_tracker("Validating Virtual Network Permissions")

    @prog
    def _validate_vnet(cmd, namespace):
        errors = []

        vnet = getattr(namespace, key)

        if not is_valid_resource_id(vnet):
            raise RequiredArgumentMissingError(
                f"Must specify --vnet if --{key.replace('_', '-')} is not an id.")

        validate_vnet(cmd, namespace)

        parts, network_client, auth_client = get_clients(vnet, cmd)

        try:
            network_client.virtual_networks.get(parts['resource_group'], parts['name'])
        except ResourceNotFoundError as err:
            raise InvalidArgumentValueError(
                f"Invalid --{key.replace('_', '-')}, error when getting '{vnet}': {str(err)}") from err

        except Exception as err:
            raise CLIInternalError(
                f"Unexpected error when getting vnet '{vnet}': {str(err)}") from err

        errors = validate_resource(auth_client, key, parts, [
            "Microsoft.Network/virtualNetworks/join/action",
            "Microsoft.Network/virtualNetworks/read",
            "Microsoft.Network/virtualNetworks/write",
            "Microsoft.Network/virtualNetworks/subnets/join/action",
            "Microsoft.Network/virtualNetworks/subnets/read",
            "Microsoft.Network/virtualNetworks/subnets/write", ])

        return errors

    return _validate_vnet


# Validating that the route tables attached to the subnet have the
# correct permissions and that the subnet is not assigned to an NSG
def dyn_validate_subnet_and_route_tables(key):
    prog = get_progress_tracker(f"Validating {key} permissions")

    @prog
    def _validate_subnet(cmd, namespace):
        errors = []

        subnet = getattr(namespace, key)

        if not is_valid_resource_id(subnet):
            if not namespace.vnet:
                raise RequiredArgumentMissingError(
                    f"Must specify --vnet if --{key.replace('_', '-')} is not an id.")

            validate_vnet(cmd, namespace)

            subnet = namespace.vnet + '/subnets/' + subnet
            setattr(namespace, key, subnet)

        parts, network_client, auth_client = get_clients(subnet, cmd)

        try:
            subnet_obj = network_client.subnets.get(parts['resource_group'],
                                                    parts['name'],
                                                    parts['child_name_1'])

            route_table_obj = subnet_obj.route_table
            if route_table_obj is not None:
                route_parts = parse_resource_id(route_table_obj.id)

                errors = validate_resource(auth_client, f"{key}_route_table", route_parts, [
                    "Microsoft.Network/routeTables/join/action",
                    "Microsoft.Network/routeTables/read",
                    "Microsoft.Network/routeTables/write"])
        except ResourceNotFoundError as err:
            raise InvalidArgumentValueError(
                f"Invalid -- subnet, error when getting '{subnet}': {str(err)}") from err

        except Exception as err:
            raise CLIInternalError(
                f"Unexpected error when getting subnet '{subnet}': {str(err)}") from err

        if subnet_obj.network_security_group is not None:
            message = f"A Network Security Group \"{subnet_obj.network_security_group.id}\" "\
                      "is already assigned to this subnet. Ensure there are no Network "\
                      "Security Groups assigned to cluster subnets before cluster creation"
            error = [key, parts['child_name_1'], message]
            errors.append(error)

        return errors

    return _validate_subnet


# Validating that the cidr ranges between the master_subnet, worker_subnet,
# service_cidr and pod_cidr do not overlap at all
def dyn_validate_cidr_ranges():
    prog = get_progress_tracker("Validating no overlapping CIDR Ranges on subnets")

    @prog
    def _validate_cidr_ranges(cmd, namespace):
        MIN_CIDR_PREFIX = 23

        addresses = []

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
            node_mask = MIN_CIDR_PREFIX - int(pod_cidr.split("/")[1])
            if node_mask < 2:
                addresses.append(["Pod CIDR",
                                  "Pod CIDR Capacity",
                                  f"{pod_cidr} does not contain enough addresses for 3 master nodes " +
                                  "(Requires cidr prefix of 21 or lower)"])
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
    prog = get_progress_tracker("Validating resource permissions")

    @prog
    def _validate_resource_permissions(cmd,
                                       _namespace):
        errors = []

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
                                           f"Resource {parts['name']} is missing role assignment " +
                                           f"{role} for service principal {sp_id} " +
                                           "(These roles will be automatically added during cluster creation)"])
                    except CloudError as e:
                        logger.error(e.message)
                        raise
        return errors
    return _validate_resource_permissions


def dyn_validate_version():
    prog = get_progress_tracker("Validating OpenShift Version")

    @prog
    def _validate_version(cmd,
                          namespace):
        errors = []

        if namespace.location is None:
            get_default_location_from_resource_group(cmd, namespace)

        versions = azext_aro.custom.aro_get_versions(namespace.client, namespace.location)

        found = False
        for version in versions:
            if version == namespace.version:
                found = True
                break

        if not found:
            errors.append(["OpenShift Version",
                           namespace.version,
                           f"{namespace.version} is not a valid version, valid versions are {versions}"])

        return errors
    return _validate_version


def validate_cluster_create(version,
                            resources,
                            service_principle_ids):
    error_object = []

    error_object.append(dyn_validate_vnet("vnet"))
    error_object.append(dyn_validate_subnet_and_route_tables("master_subnet"))
    error_object.append(dyn_validate_subnet_and_route_tables("worker_subnet"))
    error_object.append(dyn_validate_cidr_ranges())
    error_object.append(dyn_validate_resource_permissions(service_principle_ids, resources))
    if version is not None:
        error_object.append(dyn_validate_version())

    return error_object
