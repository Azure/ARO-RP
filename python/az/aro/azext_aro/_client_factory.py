import urllib3

from azext_aro.custom import rp_mode_development
from azext_aro.vendored_sdks.azure.mgmt.redhatopenshift.v2019_12_31_preview import AzureRedHatOpenShiftClient
from azure.cli.core.commands.client_factory import get_mgmt_service_client


def cf_aro(cli_ctx, *_):
    client = get_mgmt_service_client(
        cli_ctx, AzureRedHatOpenShiftClient).open_shift_clusters

    if rp_mode_development():
        client.config.base_url = "https://localhost:8443/"
        client.config.connection.verify = False
        urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

    return client
