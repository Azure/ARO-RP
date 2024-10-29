# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

import argparse

from azext_aro.vendored_sdks.azure.mgmt.redhatopenshift.v2024_08_12_preview.models import PlatformWorkloadIdentity
from azure.cli.core.azclierror import CLIError


# pylint:disable=protected-access
# pylint:disable=too-few-public-methods
class AROPlatformWorkloadIdentityAddAction(argparse._AppendAction):

    def __call__(self, parser, namespace, values, option_string=None):
        try:
            if len(values) != 2:
                msg = f"{option_string} requires 2 values in format: `OPERATOR_NAME RESOURCE_ID`"
                raise argparse.ArgumentError(self, msg)

            operator_name, resource_id = values
            parsed = (operator_name, PlatformWorkloadIdentity(resource_id=resource_id))

            super().__call__(parser, namespace, parsed, option_string)

        except ValueError as e:
            raise CLIError(f"usage error: {option_string} NAME ID") from e
