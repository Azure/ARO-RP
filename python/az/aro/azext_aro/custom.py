import os

import azext_aro.vendored_sdks.azure.mgmt.redhatopenshift.v2019_12_31_preview.models as v2019_12_31_preview

from azext_aro._aad import AADManager
from azext_aro._rbac import assign_contributor_to_vnet
from azext_aro._validators import validate_subnets
from azure.cli.core.commands.client_factory import get_subscription_id
from azure.cli.core.util import sdk_no_wait
from msrestazure.azure_exceptions import CloudError


FP_CLIENT_ID = "f1dd0a37-89c6-4e07-bcd1-ffd3d43d8875"


def aro_create(cmd,  # pylint: disable=too-many-locals
               client,
               resource_group_name,
               resource_name,
               master_subnet,
               worker_subnet,
               vnet=None,
               vnet_resource_group_name=None,  # pylint: disable=unused-argument
               location=None,
               client_id=None,
               client_secret=None,
               pod_cidr=None,
               service_cidr=None,
               master_vm_size=None,
               worker_vm_size=None,
               worker_vm_disk_size_gb=None,
               worker_count=None,
               tags=None,
               no_wait=False):
    vnet = validate_subnets(master_subnet, worker_subnet)

    subscription_id = get_subscription_id(cmd.cli_ctx)

    aad = AADManager(cmd.cli_ctx)
    if client_id is None:
        app, client_secret = aad.createManagedApplication(
            "aro-%s-%s-%s" % (subscription_id, resource_group_name, resource_name))
        client_id = app.app_id

    client_sp = aad.getServicePrincipal(client_id)
    if not client_sp:
        client_sp = aad.createServicePrincipal(client_id)

    rp_client_id = FP_CLIENT_ID
    if rp_mode_development():
        rp_client_id = os.environ['AZURE_FP_CLIENT_ID']

    rp_client_sp = aad.getServicePrincipal(rp_client_id)

    assign_contributor_to_vnet(cmd.cli_ctx, vnet, client_sp.object_id)
    assign_contributor_to_vnet(cmd.cli_ctx, vnet, rp_client_sp.object_id)

    oc = v2019_12_31_preview.OpenShiftCluster(
        location=location,
        tags=tags,
        service_principal_profile=v2019_12_31_preview.ServicePrincipalProfile(
            client_id=client_id,
            client_secret=client_secret,
        ),
        network_profile=v2019_12_31_preview.NetworkProfile(
            pod_cidr=pod_cidr or "10.128.0.0/14",
            service_cidr=service_cidr or "172.30.0.0/16",
        ),
        master_profile=v2019_12_31_preview.MasterProfile(
            vm_size=master_vm_size or "Standard_D8s_v3",
            subnet_id=master_subnet,
        ),
        worker_profiles=[
            v2019_12_31_preview.WorkerProfile(
                name="worker",  # TODO: "worker" should not be hard-coded
                vm_size=worker_vm_size or "Standard_D2s_v3",
                disk_size_gb=worker_vm_disk_size_gb or 128,
                subnet_id=worker_subnet,
                count=worker_count or 3,
            )
        ]
    )

    return sdk_no_wait(no_wait, client.create_or_update,
                       resource_group_name=resource_group_name,
                       resource_name=resource_name,
                       parameters=oc)


def aro_delete(cmd, client, resource_group_name, resource_name,
               no_wait=False):
    if not no_wait:
        aad = AADManager(cmd.cli_ctx)
        try:
            oc = client.get(resource_group_name, resource_name)
        except CloudError as err:
            if err.status_code == 404:
                return
            raise err

    sdk_no_wait(no_wait, client.delete,
                resource_group_name=resource_group_name,
                resource_name=resource_name)

    if not no_wait:
        aad.deleteManagedApplication(oc.service_principal_profile.client_id)


def aro_list(client, resource_group_name=None):
    if resource_group_name:
        return client.list_by_resource_group(resource_group_name).value
    return client.list().value


def aro_show(client, resource_group_name, resource_name):
    return client.get(resource_group_name, resource_name)


def aro_get_credentials(client, resource_group_name, resource_name):
    return client.get_credentials(resource_group_name, resource_name)


def aro_update(client, resource_group_name, resource_name,
               worker_count=None,
               no_wait=False):
    oc = client.get(resource_group_name, resource_name)

    if worker_count is not None:
        # TODO: [0] should not be hard-coded
        oc.worker_profiles[0].count = worker_count

    return sdk_no_wait(no_wait, client.create_or_update,
                       resource_group_name=resource_group_name,
                       resource_name=resource_name,
                       parameters=oc)


def rp_mode_development():
    return os.environ.get('RP_MODE', '').lower() == 'development'
