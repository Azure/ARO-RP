# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

import collections

from msrestazure.tools import parse_resource_id


def aro_list_table_format(results):
    return [aro_show_table_format(r) for r in results]


def aro_show_table_format(result):
    parts = parse_resource_id(result["id"])

    return collections.OrderedDict(
        Name=result["name"],
        ResourceGroup=parts["resource_group"],
        Location=result["location"],
        ProvisioningState=result["provisioningState"],
        WorkerCount=result["workerProfiles"][0]["count"],
        ConsoleURL=result["consoleUrl"],
    )
