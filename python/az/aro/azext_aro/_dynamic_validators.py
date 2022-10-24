# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

import ipaddress
import json
import re
import uuid

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
from azext_aro._validators import validate_vnet, validate_vnet_resource_group_name


logger = get_logger(__name__)


def dyn_validate_subnet(key):
    def _validate_subnet(cmd, namespace):
        subnet = getattr(namespace, key)

        if not is_valid_resource_id(subnet):
            if not namespace.vnet:
                raise RequiredArgumentMissingError(
                    f"Must specify --vnet if --{key.replace('_', '-')} is not an id.")

            validate_vnet(cmd, namespace)

            subnet = namespace.vnet + '/subnets/' + subnet
            setattr(namespace, key, subnet)

        parts = parse_resource_id(subnet)

        client = get_mgmt_service_client(
            cmd.cli_ctx, ResourceType.MGMT_NETWORK)

        return 'hello'

        # try:
        #     client.subnets.get(parts['resource_group'],
        #                        parts['name'], parts['child_name_1'])
        # except Exception as err:
        #     if isinstance(err, ResourceNotFoundError):
        #         raise InvalidArgumentValueError(
        #             f"Invalid --{key.replace('_', '-')}, error when getting '{subnet}': {str(err)}") from err
        #     raise CLIInternalError(
        #         f"Unexpected error when getting subnet '{subnet}': {str(err)}") from err

    return _validate_subnet


def validate_cluster_create(cmd, client, oc, resource_group_name, master_subnet, worker_subnet, vnet):
    error_object = {}

    error_object['master_subnet_validation'] = dyn_validate_subnet(
        master_subnet)
    error_object['worker_subnet_validation'] = dyn_validate_subnet(
        worker_subnet)

    return error_object
