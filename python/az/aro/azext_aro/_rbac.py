# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

import os
import uuid

from azure.cli.core.commands.client_factory import get_mgmt_service_client
from azure.cli.core.commands.client_factory import get_subscription_id
from azure.cli.core.profiles import get_sdk
from azure.cli.core.profiles import ResourceType
from msrestazure.tools import resource_id
from msrestazure.tools import parse_resource_id


CONTRIBUTOR = 'b24988ac-6180-42a0-ab88-20f7382dd24c'
DEVELOPMENT_CONTRIBUTOR = 'f3fe7bc1-0ef9-4681-a68c-c1fa285d6128'


def _gen_uuid():
    return uuid.uuid4()


def assign_contributor_to_vnet(cli_ctx, vnet, object_id):
    auth_client = get_mgmt_service_client(cli_ctx, ResourceType.MGMT_AUTHORIZATION)

    RoleAssignmentCreateParameters = get_sdk(cli_ctx, ResourceType.MGMT_AUTHORIZATION,
                                             'RoleAssignmentCreateParameters', mod='models',
                                             operation_group='role_assignments')

    role_definition_id = resource_id(
        subscription=get_subscription_id(cli_ctx),
        namespace='Microsoft.Authorization',
        type='roleDefinitions',
        name=DEVELOPMENT_CONTRIBUTOR if rp_mode_development() else CONTRIBUTOR,
    )

    if has_assignment(auth_client.role_assignments.list_for_scope(vnet), role_definition_id, object_id):
        return

    # generate random uuid for role assignment
    role_uuid = _gen_uuid()

    auth_client.role_assignments.create(vnet, role_uuid, RoleAssignmentCreateParameters(
        role_definition_id=role_definition_id,
        principal_id=object_id,
        principal_type='ServicePrincipal',
    ))


def assign_contributor_to_routetable(cli_ctx, master_subnet, worker_subnet, object_id):
    auth_client = get_mgmt_service_client(cli_ctx, ResourceType.MGMT_AUTHORIZATION)
    network_client = get_mgmt_service_client(cli_ctx, ResourceType.MGMT_NETWORK)

    RoleAssignmentCreateParameters = get_sdk(cli_ctx, ResourceType.MGMT_AUTHORIZATION,
                                             'RoleAssignmentCreateParameters', mod='models',
                                             operation_group='role_assignments')

    role_definition_id = resource_id(
        subscription=get_subscription_id(cli_ctx),
        namespace='Microsoft.Authorization',
        type='roleDefinitions',
        name=DEVELOPMENT_CONTRIBUTOR if rp_mode_development() else CONTRIBUTOR,
    )

    route_tables = set()
    for sn in [master_subnet, worker_subnet]:
        sid = parse_resource_id(sn)

        subnet = network_client.subnets.get(resource_group_name=sid['resource_group'],
                                            virtual_network_name=sid['name'],
                                            subnet_name=sid['resource_name'])

        if subnet.route_table is not None:
            route_tables.add(subnet.route_table.id)

    for rt in route_tables:
        if has_assignment(auth_client.role_assignments.list_for_scope(subnet.route_table.id),
                          role_definition_id, object_id):
            continue

        role_uuid = _gen_uuid()

        auth_client.role_assignments.create(rt, role_uuid, RoleAssignmentCreateParameters(
            role_definition_id=role_definition_id,
            principal_id=object_id,
            principal_type='ServicePrincipal',
        ))


def has_assignment(assignments, role_definition_id, object_id):
    for assignment in assignments:
        if assignment.role_definition_id.lower() == role_definition_id.lower() and \
                assignment.principal_id.lower() == object_id.lower():
            return True

    return False


def rp_mode_development():
    return os.environ.get('RP_MODE', '').lower() == 'development'
