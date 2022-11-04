# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

import ipaddress
import json
import re
from unicodedata import name
import uuid
import collections
from itertools import tee

from azure.cli.core.commands.client_factory import get_mgmt_service_client
from azure.cli.core.commands.client_factory import get_subscription_id
from azure.cli.core.profiles import ResourceType
from azure.cli.core.azclierror import CLIInternalError, InvalidArgumentValueError, \
    RequiredArgumentMissingError
from azure.core.exceptions import ResourceNotFoundError
from knack.log import get_logger
from msrestazure.azure_exceptions import CloudError
from msrestazure.tools import is_valid_resource_id
from msrestazure.tools import parse_resource_id
from msrestazure.tools import resource_id
from azext_aro._validators import validate_vnet, validate_vnet_resource_group_name, validate_cidr


logger = get_logger(__name__)

def can_do_action(perms, action):
    for perm in perms:
        for perm_action in perm.actions:
            clean = re.escape(perm_action)
            clean = re.match("(?i)^" + clean.replace("\*", ".*") + "$", action)
            if clean:
                return None
        for not_action in perm.not_actions:
            clean = re.escape(not_action)
            clean = re.match("(?i)^" + clean.replace("\*", ".*") + "$", action)
            if clean:
                return f"{action} permission is missing"

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
        error = can_do_action(perms_copy, action)
        if error != None:
            errors.append(f"{key} -- {error}")

    return errors

def dyn_validate_vnet(key):
    def _validate_vnet(cmd, namespace):
        vnet = namespace[key]

        if not is_valid_resource_id(vnet):
            raise RequiredArgumentMissingError(
                f"Must specify --vnet if --{key.replace('_', '-')} is not an id.")

        namespace_obj = collections.namedtuple("Namespace", namespace.keys())(*namespace.values())

        validate_vnet(cmd, namespace_obj)

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

        return validate_resource(auth_client, key, parts, [
            "Microsoft.Network/virtualNetworks/join/action",
            "Microsoft.Network/virtualNetworks/read",
            "Microsoft.Network/virtualNetworks/write",
            "Microsoft.Network/virtualNetworks/subnets/join/action",
            "Microsoft.Network/virtualNetworks/subnets/read",
            "Microsoft.Network/virtualNetworks/subnets/write",])

    return _validate_vnet

def dyn_validate_subnet(key):
    def _validate_subnet(cmd, namespace):
        subnet = namespace[key]

        if not is_valid_resource_id(subnet):
            if not namespace["vnet"]:
                raise RequiredArgumentMissingError(
                    f"Must specify --vnet if --{key.replace('_', '-')} is not an id.")

            validate_vnet(cmd, namespace)

            subnet = namespace["vnet"] + '/subnets/' + subnet
            setattr(namespace, key, subnet)

        parts = parse_resource_id(subnet)

        network_client = get_mgmt_service_client(
            cmd.cli_ctx, ResourceType.MGMT_NETWORK)

        auth_client = get_mgmt_service_client(
            cmd.cli_ctx, ResourceType.MGMT_AUTHORIZATION, api_version="2015-07-01")

        subnet_obj = None
        route_table_obj = None

        try:
            subnet_obj = network_client.subnets.get(parts['resource_group'],
                               parts['name'], parts['child_name_1'])
            route_table_obj = subnet_obj.route_table
        except Exception as err:
            if isinstance(err, ResourceNotFoundError):
                raise InvalidArgumentValueError(
                    f"Invalid --{key.replace('_', '-')}, error when getting '{subnet}': {str(err)}") from err
            raise CLIInternalError(
                f"Unexpected error when getting subnet '{subnet}': {str(err)}") from err

        route_parts = parse_resource_id(route_table_obj.id)

        return validate_resource(auth_client, key, route_parts, [
            "Microsoft.Network/routeTables/join/action",
            "Microsoft.Network/routeTables/read",
            "Microsoft.Network/routeTables/write",])

    return _validate_subnet

def dyn_validate_cidr_ranges():
    def _validate_cidr_ranges(cmd, namespace):
        vnet = namespace["vnet"]
        master_subnet = namespace["master_subnet"]
        worker_subnet = namespace["worker_subnet"]
        pod_cidr = namespace["pod_cidr"]
        service_cidr = namespace["service_cidr"]

        vnet_parts = parse_resource_id(vnet)
        worker_parts = parse_resource_id(worker_subnet)
        master_parts = parse_resource_id(master_subnet)

        validate_cidr(pod_cidr)
        validate_cidr(service_cidr)

        cidr_array = {}

        if pod_cidr != None:
            cidr_array["Pod CIDR"] = ipaddress.IPv4Network(pod_cidr)
        if service_cidr != None:
            cidr_array["Service CIDR"] = ipaddress.IPv4Network(service_cidr)

        worker_subnet_obj = None
        master_subnet_obj = None

        network_client = get_mgmt_service_client(
            cmd.cli_ctx, ResourceType.MGMT_NETWORK)

        try:
            worker_subnet_obj = network_client.subnets.get(vnet_parts['resource_group'],
                               vnet_parts['name'], worker_parts['child_name_1'])
        except Exception as err:
            if isinstance(err, ResourceNotFoundError):
                raise InvalidArgumentValueError(
                    f"Invalid -- worker_subnet, error when getting '{worker_subnet}': {str(err)}") from err
            raise CLIInternalError(
                f"Unexpected error when getting subnet '{worker_subnet}': {str(err)}") from err

        if worker_subnet_obj.address_prefix == None:
            for address in worker_subnet_obj.address_prefixes:
                cidr_array["Worker Subnet CIDR -- " + address] = ipaddress.IPv4Network(address)
        else:
            cidr_array["Worker Subnet CIDR"] = ipaddress.IPv4Network(worker_subnet_obj.address_prefix)


        try:
            master_subnet_obj = network_client.subnets.get(vnet_parts['resource_group'],
                               vnet_parts['name'], master_parts['child_name_1'])
        except Exception as err:
            if isinstance(err, ResourceNotFoundError):
                raise InvalidArgumentValueError(
                    f"Invalid -- master_subnet, error when getting '{master_subnet}': {str(err)}") from err
            raise CLIInternalError(
                f"Unexpected error when getting subnet '{master_subnet}': {str(err)}") from err

        if master_subnet_obj.address_prefix == None:
            for address in master_subnet_obj.address_prefixes:
                cidr_array["Master Subnet CIDR -- " + address] = ipaddress.IPv4Network(address)
        else:
            cidr_array["Master Subnet CIDR"] = ipaddress.IPv4Network(master_subnet_obj.address_prefix)

        ipv4_zero = ipaddress.IPv4Network("0.0.0.0/0")

        addresses = []

        for key in cidr_array:
            cidr = cidr_array[key]
            if not cidr.overlaps(ipv4_zero):
                error = f"{key} -- CIDR {cidr} is not valid as it does not overlap with {ipv4_zero}"
                addresses.append(error)
            for key2 in cidr_array:
                compare = cidr_array[key2]
                if cidr != compare:
                    if cidr.overlaps(compare):
                        error = f"{key} -- CIDR {cidr} is not valid as it overlaps with {compare}"
                        addresses.append(cidr)

        return addresses

    return _validate_cidr_ranges

def validate_cluster_create(cmd, client, resource_group_name, master_subnet, worker_subnet, vnet, pod_cidr, service_cidr):
    error_object = {}

    error_object['vnet_validation'] = dyn_validate_vnet("vnet")
    error_object['master_subnet_validation'] = dyn_validate_subnet("master_subnet")
    error_object['worker_subnet_validation'] = dyn_validate_subnet("worker_subnet")
    error_object['cidr_range_validation'] = dyn_validate_cidr_ranges()

    return error_object