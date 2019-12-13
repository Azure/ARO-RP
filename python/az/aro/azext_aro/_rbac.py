import uuid

from azure.cli.core.commands.client_factory import get_mgmt_service_client
from azure.cli.core.commands.client_factory import get_subscription_id
from azure.cli.core.profiles import get_sdk
from azure.cli.core.profiles import ResourceType
from msrestazure.azure_exceptions import CloudError
from msrestazure.tools import resource_id


CONTRIBUTOR = "b24988ac-6180-42a0-ab88-20f7382dd24c"


def assign_contributor_to_vnet(cli_ctx, vnet, object_id):
    client = get_mgmt_service_client(cli_ctx, ResourceType.MGMT_AUTHORIZATION)

    RoleAssignmentCreateParameters = get_sdk(cli_ctx, ResourceType.MGMT_AUTHORIZATION,
                                             'RoleAssignmentCreateParameters', mod='models',
                                             operation_group='role_assignments')

    try:
        client.role_assignments.create(vnet, uuid.uuid4(), RoleAssignmentCreateParameters(
            role_definition_id=resource_id(
                subscription=get_subscription_id(cli_ctx),
                namespace='Microsoft.Authorization',
                type='roleDefinitions',
                name=CONTRIBUTOR,
            ),
            principal_id=object_id,
            principal_type="ServicePrincipal",
        ))
    except CloudError as err:
        if err.status_code == 409:
            return
        raise err
