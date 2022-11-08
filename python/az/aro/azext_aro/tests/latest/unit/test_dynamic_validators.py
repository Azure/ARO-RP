# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

from unittest.mock import Mock, patch
from azext_aro._dynamic_validators import (
    dyn_validate_cidr_ranges
)
import pytest


test_validate_cidr_data = [
    (
        "should return no error on address_prefix set on subnets, no additional cidrs, no overlap",
        Mock(cli_ctx=None),
        Mock(vnet='', key="192.168.0.1/32", master_subnet='', worker_subnet='', pod_cidr=None, service_cidr=None),
        Mock(**{"subnets.get.side_effect": [Mock(address_prefix="172.143.5.0/24"), Mock(address_prefix="172.143.4.0/25")]}),
        {
            "subscription": "subscription",
            "namespace": "MICROSOFT.NETWORK",
            "type": "virtualnetworks",
            "last_child_num": 1,
            "child_type_1": "subnets",
            "resource_group": None,
            "name": None,
            "child_name_1": None
        },
        None,
        None
    ),
    (
        "should return no error on address_prefix set on subnets, additional cidrs, no overlap",
        Mock(cli_ctx=None),
        Mock(vnet='', key="192.168.0.1/32", master_subnet='', worker_subnet='', pod_cidr="172.143.7.0/24", service_cidr="172.143.6.0/25"),
        Mock(**{"subnets.get.side_effect": [Mock(address_prefix="172.143.4.0/24"), Mock(address_prefix="172.143.5.0/25")]}),
        {
            "subscription": "subscription",
            "namespace": "MICROSOFT.NETWORK",
            "type": "virtualnetworks",
            "last_child_num": 1,
            "child_type_1": "subnets",
            "resource_group": None,
            "name": None,
            "child_name_1": None
        },
        None,
        None
    ),
        (
        "should return no error on address_prefixes set on subnets, additional cidrs, no overlap",
        Mock(cli_ctx=None),
        Mock(vnet='', key="192.168.0.1/32", master_subnet='', worker_subnet='', pod_cidr="172.143.7.0/24", service_cidr="172.143.6.0/25"),
        Mock(**{"subnets.get.side_effect": [Mock(address_prefix=None, 
                                                 address_prefixes=["172.143.4.0/24", "172.143.8.0/25"]),
                                            Mock(address_prefix=None,
                                                 address_prefixes=["172.143.5.0/25", "172.143.9.0/24"])]}),
        {
            "subscription": "subscription",
            "namespace": "MICROSOFT.NETWORK",
            "type": "virtualnetworks",
            "last_child_num": 1,
            "child_type_1": "subnets",
            "resource_group": None,
            "name": None,
            "child_name_1": None
        },
        None,
        None
    ),
    (
        "should error on address_prefix set on subnets, no additional cidrs, overlap",
        Mock(cli_ctx=None),
        Mock(vnet='', key="192.168.0.1/32", master_subnet='', worker_subnet='', pod_cidr=None, service_cidr=None),
        Mock(**{"subnets.get.side_effect": [Mock(address_prefix="172.143.4.0/24"), Mock(address_prefix="172.143.4.0/25")]}),
        {
            "subscription": "subscription",
            "namespace": "MICROSOFT.NETWORK",
            "type": "virtualnetworks",
            "last_child_num": 1,
            "child_type_1": "subnets",
            "resource_group": None,
            "name": None,
            "child_name_1": None
        },
        ["Master Subnet CIDR -- CIDR 172.143.4.0/24 is not valid as it overlaps with 172.143.4.0/25"],
        None
    ),
]


@pytest.mark.parametrize(
    "test_description, cmd_mock, namespace_mock, client_mock, parse_resource_id_mock_return_value, expected_addresses, expected_exception",
    test_validate_cidr_data,
    ids=[i[0] for i in test_validate_cidr_data]
)
@ patch('azext_aro._dynamic_validators.get_mgmt_service_client')
@ patch('azext_aro._dynamic_validators.parse_resource_id')
def test_validate_cidr(
    # Mocked functions:
    parse_resource_id_mock, get_mgmt_service_client_mock, 

    # Test cases parameters:
    test_description, cmd_mock, namespace_mock, client_mock, parse_resource_id_mock_return_value, expected_addresses, expected_exception
):
    parse_resource_id_mock.return_value = parse_resource_id_mock_return_value
    get_mgmt_service_client_mock.return_value = client_mock

    validate_cidr_fn = dyn_validate_cidr_ranges()
    if expected_exception is None and expected_addresses is None:
        addresses = validate_cidr_fn(cmd_mock, namespace_mock)

        if (len(addresses) > 0):
            raise Exception(f"Unexpected Error: {addresses[0]}")
    elif(expected_exception is None):
        addresses = validate_cidr_fn(cmd_mock, namespace_mock)

        if (addresses[0] != expected_addresses[0]):
            raise Exception(f"Error returned was not expected\n Expected : {expected_addresses[0]}\n Actual   : {addresses[0]}")
    else:
        with pytest.raises(expected_exception):
            validate_cidr_fn(cmd_mock, namespace_mock)
