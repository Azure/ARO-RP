# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

from unittest.mock import Mock, patch
from azext_aro._dynamic_validators import (
    dyn_validate_cidr_ranges, dyn_validate_subnet, dyn_validate_vnet, dyn_validate_resource_permissions
)

from azure.mgmt.authorization.models import Permission
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
        None
    ),
    (
        "should return no error on address_prefix set on subnets, additional cidrs, no overlap",
        Mock(cli_ctx=None),
        Mock(vnet='', key="192.168.0.1/32", master_subnet='', worker_subnet='', pod_cidr="172.142.0.0/21", service_cidr="172.143.6.0/25"),
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
        None
    ),
    (
        "should return no error on multiple address_prefixes set on subnets, additional cidrs, no overlap",
        Mock(cli_ctx=None),
        Mock(vnet='', key="192.168.0.1/32", master_subnet='', worker_subnet='', pod_cidr="172.142.0.0/21", service_cidr="172.143.6.0/25"),
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
        "172.143.4.0/24 is not valid as it overlaps with 172.143.4.0/25"
    ),
    (
        "should return error on pod cidr not having enough addresses to create cluster",
        Mock(cli_ctx=None),
        Mock(vnet='', key="192.168.0.1/32", master_subnet='', worker_subnet='', pod_cidr="172.143.4.0/24", service_cidr=None),
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
        "172.143.4.0/24 does not contain enough addresses for 3 master nodes (Requires cidr prefix of 21 or lower)"
    ),
]


@pytest.mark.parametrize(
    "test_description, cmd_mock, namespace_mock, client_mock, parse_resource_id_mock_return_value, expected_addresses",
    test_validate_cidr_data,
    ids=[i[0] for i in test_validate_cidr_data]
)
@ patch('azext_aro._dynamic_validators.get_mgmt_service_client')
@ patch('azext_aro._dynamic_validators.parse_resource_id')
def test_validate_cidr(
    # Mocked functions:
    parse_resource_id_mock, get_mgmt_service_client_mock,

    # Test cases parameters:
    test_description, cmd_mock, namespace_mock, client_mock, parse_resource_id_mock_return_value, expected_addresses
):
    parse_resource_id_mock.return_value = parse_resource_id_mock_return_value
    get_mgmt_service_client_mock.return_value = client_mock

    validate_cidr_fn = dyn_validate_cidr_ranges()
    if expected_addresses is None:
        addresses = validate_cidr_fn(cmd_mock, namespace_mock)

        if (len(addresses) > 0):
            raise Exception(f"Unexpected Error: {addresses[0]}")
    else:
        addresses = validate_cidr_fn(cmd_mock, namespace_mock)

        if (addresses[0][2] != expected_addresses):
            raise Exception(f"Error returned was not expected\n Expected : {expected_addresses}\n Actual   : {addresses[0][2]}")


test_validate_subnets_data = [
    (
        "should not return missing permission when actions are permitted",
        Mock(cli_ctx=None),
        Mock(vnet='', key="192.168.0.1/32", master_subnet='', worker_subnet='', pod_cidr=None, service_cidr=None),
        Mock(**{"subnets.get.return_value": Mock(network_security_group=None, route_table=Mock(id="test"))}),
        Mock(**{"permissions.list_for_resource.return_value": [Permission(actions=["Microsoft.Network/routeTables/*"], not_actions=[])]}),
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
        Mock(),
        None
    ),
    (
        "should return missing permission when actions are not permitted",
        Mock(cli_ctx=None),
        Mock(vnet='', key="192.168.0.1/32", master_subnet='', worker_subnet='', pod_cidr=None, service_cidr=None),
        Mock(**{"subnets.get.return_value": Mock(network_security_group=None, route_table=Mock(id="test"))}),
        Mock(**{"permissions.list_for_resource.return_value": [Permission(actions=[], not_actions=["Microsoft.Network/routeTables/*"])]}),
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
        Mock(),
        "Microsoft.Network/routeTables/join/action permission is disabled"
    ),
    (
        "should return missing permission when actions are not present",
        Mock(cli_ctx=None),
        Mock(vnet='', key="192.168.0.1/32", master_subnet='', worker_subnet='', pod_cidr=None, service_cidr=None),
        Mock(**{"subnets.get.return_value": Mock(network_security_group=None, route_table=Mock(id="test"))}),
        Mock(**{"permissions.list_for_resource.return_value": [Permission(actions=[], not_actions=[])]}),
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
        Mock(),
        "Microsoft.Network/routeTables/join/action permission is missing"
    ),
    (
        "should return message when network security group is already attached to subnet",
        Mock(cli_ctx=None),
        Mock(vnet='', key="192.168.0.1/32", master_subnet='', worker_subnet='', pod_cidr=None, service_cidr=None),
        Mock(**{"subnets.get.return_value": Mock(network_security_group=Mock(id="test"), route_table=Mock(id="test"))}),
        Mock(**{"permissions.list_for_resource.return_value": [Permission(actions=["Microsoft.Network/routeTables/*"], not_actions=[])]}),
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
        Mock(),
        "A Network Security Group \"test\" is already assigned to this subnet. "
        "Ensure there a no Network Security Groups assigned to cluster "
        "subnets before cluster creation"
    )
]


@pytest.mark.parametrize(
    "test_description, cmd_mock, namespace_mock, network_client_mock, auth_client_mock, parse_resource_id_mock_return_value, is_valid_resource_id_mock_return_value, expected_missing_perms",
    test_validate_subnets_data,
    ids=[i[0] for i in test_validate_subnets_data]
)
@ patch('azext_aro._dynamic_validators.get_mgmt_service_client')
@ patch('azext_aro._dynamic_validators.parse_resource_id')
@ patch('azext_aro._dynamic_validators.is_valid_resource_id')
def test_validate_subnets(
    # Mocked functions:
    is_valid_resource_id_mock, parse_resource_id_mock, get_mgmt_service_client_mock,

    # Test cases parameters:
    test_description, cmd_mock, namespace_mock, network_client_mock, auth_client_mock, parse_resource_id_mock_return_value, is_valid_resource_id_mock_return_value, expected_missing_perms
):
    is_valid_resource_id_mock.return_value = is_valid_resource_id_mock_return_value
    parse_resource_id_mock.return_value = parse_resource_id_mock_return_value
    get_mgmt_service_client_mock.side_effect = [network_client_mock, auth_client_mock]

    validate_subnet_fn = dyn_validate_subnet('')
    if expected_missing_perms is None:
        missing_perms = validate_subnet_fn(cmd_mock, namespace_mock)

        if (len(missing_perms) > 0):
            raise Exception(f"Unexpected Permission Missing: {missing_perms[0]}")
    else:
        missing_perms = validate_subnet_fn(cmd_mock, namespace_mock)

        if (missing_perms[0][2] != expected_missing_perms):
            raise Exception(f"Error returned was not expected\n Expected : {expected_missing_perms}\n Actual   : {missing_perms[0][2]}")


test_validate_vnets_data = [
    (
        "should not return missing permission when actions are permitted",
        Mock(cli_ctx=None),
        Mock(vnet='', key="192.168.0.1/32", master_subnet='', worker_subnet='', pod_cidr=None, service_cidr=None),
        Mock(**{"subnets.get.return_value": Mock(route_table=Mock(id="test"))}),
        Mock(**{"permissions.list_for_resource.return_value": [Permission(actions=["Microsoft.Network/virtualNetworks/*"], not_actions=[])]}),
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
        Mock(),
        None
    ),
    (
        "should return disabled permission when actions are not permitted",
        Mock(cli_ctx=None),
        Mock(vnet='', key="192.168.0.1/32", master_subnet='', worker_subnet='', pod_cidr=None, service_cidr=None),
        Mock(**{"subnets.get.return_value": Mock(route_table=Mock(id="test"))}),
        Mock(**{"permissions.list_for_resource.return_value": [Permission(actions=[], not_actions=["Microsoft.Network/virtualNetworks/*"])]}),
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
        Mock(),
        "Microsoft.Network/virtualNetworks/join/action permission is disabled"
    ),
    (
        "should return missing permission when actions are not present",
        Mock(cli_ctx=None),
        Mock(vnet='', key="192.168.0.1/32", master_subnet='', worker_subnet='', pod_cidr=None, service_cidr=None),
        Mock(**{"subnets.get.return_value": Mock(route_table=Mock(id="test"))}),
        Mock(**{"permissions.list_for_resource.return_value": [Permission(actions=[], not_actions=[])]}),
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
        Mock(),
        "Microsoft.Network/virtualNetworks/join/action permission is missing"
    )
]


@pytest.mark.parametrize(
    "test_description, cmd_mock, namespace_mock, network_client_mock, auth_client_mock, parse_resource_id_mock_return_value, is_valid_resource_id_mock_return_value, expected_missing_perms",
    test_validate_vnets_data,
    ids=[i[0] for i in test_validate_vnets_data]
)
@ patch('azext_aro._dynamic_validators.get_mgmt_service_client')
@ patch('azext_aro._dynamic_validators.parse_resource_id')
@ patch('azext_aro._dynamic_validators.is_valid_resource_id')
def test_validate_vnets(
    # Mocked functions:
    is_valid_resource_id_mock, parse_resource_id_mock, get_mgmt_service_client_mock,

    # Test cases parameters:
    test_description, cmd_mock, namespace_mock, network_client_mock, auth_client_mock, parse_resource_id_mock_return_value, is_valid_resource_id_mock_return_value, expected_missing_perms
):
    is_valid_resource_id_mock.return_value = is_valid_resource_id_mock_return_value
    parse_resource_id_mock.return_value = parse_resource_id_mock_return_value
    get_mgmt_service_client_mock.side_effect = [network_client_mock, auth_client_mock]

    validate_vnet_fn = dyn_validate_vnet("vnet")
    if expected_missing_perms is None:
        missing_perms = validate_vnet_fn(cmd_mock, namespace_mock)

        if (len(missing_perms) > 0):
            raise Exception(f"Unexpected Permission Missing: {missing_perms[0]}")
    else:
        missing_perms = validate_vnet_fn(cmd_mock, namespace_mock)

        if (missing_perms[0][2] != expected_missing_perms):
            raise Exception(f"Error returned was not expected\n Expected : {expected_missing_perms}\n Actual   : {missing_perms[0][2]}")


test_validate_resource_data = [
    (
        "should not return missing permission when role assignments are assigned",
        Mock(cli_ctx=None),
        Mock(),
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
        [True, True, True, True, True, True, True, True],
        None
    ),
    (
        "should return missing permission when role assignments are not assigned",
        Mock(cli_ctx=None),
        Mock(),
        {
            "subscription": "subscription",
            "namespace": "MICROSOFT.NETWORK",
            "type": "virtualnetworks",
            "last_child_num": 1,
            "child_type_1": "subnets",
            "resource_group": None,
            "name": "Test_Subnet",
            "child_name_1": None
        },
        [True, True, True, True, True, True, True, False],
        "Resource Test_Subnet is missing role assignment test"
    )
]


@pytest.mark.parametrize(
    "test_description, cmd_mock, namespace_mock, parse_resource_id_mock_return_value, has_role_assignment_on_resource_mock_return_value, expected_missing_perms",
    test_validate_resource_data,
    ids=[i[0] for i in test_validate_resource_data]
)
@ patch('azext_aro._dynamic_validators.has_role_assignment_on_resource')
@ patch('azext_aro._dynamic_validators.parse_resource_id')
def test_validate_resources(
    # Mocked functions:
    parse_resource_id_mock, has_role_assignment_on_resource_mock,

    # Test cases parameters:
    test_description, cmd_mock, namespace_mock, parse_resource_id_mock_return_value, has_role_assignment_on_resource_mock_return_value, expected_missing_perms
):
    parse_resource_id_mock.return_value = parse_resource_id_mock_return_value
    has_role_assignment_on_resource_mock.side_effect = has_role_assignment_on_resource_mock_return_value

    sp_ids = ["test", "test"]
    resources = {"test": "test", "test": "test"}
    validate_res_perms_fn = dyn_validate_resource_permissions(sp_ids, resources)
    if expected_missing_perms is None:
        missing_perms = validate_res_perms_fn(cmd_mock, namespace_mock)

        if (len(missing_perms) > 0):
            raise Exception(f"Unexpected Permission Missing: {missing_perms[0]}")
    else:
        missing_perms = validate_res_perms_fn(cmd_mock, namespace_mock)

        if (missing_perms[0][2] != expected_missing_perms):
            raise Exception(f"Error returned was not expected\n Expected : {expected_missing_perms}\n Actual   : {missing_perms[0][2]}")
