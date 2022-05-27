# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

from typing import List
from unittest.mock import Mock, patch
from azext_aro._validators import (
    validate_cidr, validate_client_id, validate_client_secret, validate_cluster_resource_group,
    validate_disk_encryption_set, validate_domain, validate_pull_secret, validate_sdn, validate_subnet, validate_subnets,
    validate_visibility, validate_vnet_resource_group_name, validate_worker_count, validate_worker_vm_disk_size_gb, validate_refresh_cluster_credentials
)
from azure.cli.core.azclierror import (
    InvalidArgumentValueError, RequiredArgumentMissingError, RequiredArgumentMissingError, CLIInternalError
)
from azure.core.exceptions import ResourceNotFoundError

import pytest

test_validate_cidr_data = [
    ("should not raise exception when valid IPv4 address", Mock(key='192.168.0.0/28'), "key", None),
    ("should raise InvalidArgumentValueError when non valid IPv4 address due to beeing a simple string", Mock(key='this is an invalid network'), "key", InvalidArgumentValueError),
    ("should raise InvalidArgumentValueError when non valid IPv4 address due to invalid network ID", Mock(key='192.168.0.0.0.0/28'), "key", InvalidArgumentValueError),
    ("should raise InvalidArgumentValueError when non valid IPv4 address due to invalid hostID", Mock(key='192.168.0.0.0.0/2888'), "key", InvalidArgumentValueError),
    ("should not raise exception when IPv4 address is None", Mock(key=None), "key", None)
]


@pytest.mark.parametrize("test_description, dummyclass, attribute_to_get_from_object, expected_exception", test_validate_cidr_data, ids=[i[0] for i in test_validate_cidr_data])
def test_validate_cidr(test_description, dummyclass, attribute_to_get_from_object, expected_exception):
    validate_cidr_fn = validate_cidr(attribute_to_get_from_object)
    if expected_exception is None:
        validate_cidr_fn(dummyclass)
    else:
        with pytest.raises(expected_exception):
            validate_cidr_fn(dummyclass)


test_validate_client_id_data = [
    ("should not raise any Exception when namespace.client_id is None", Mock(client_id=None), None),
    ("should raise InvalidArgumentValueError when it can not create a UUID from namespace.client_id", Mock(client_id="invalid_client_id"), InvalidArgumentValueError),
    ("should raise RequiredArgumentMissingError when can not crate a string representation from namespace.client_secret because is None", Mock(client_id="12345678123456781234567812345678", client_secret=None), RequiredArgumentMissingError),
    ("should raise RequiredArgumentMissingError when can not crate a string representation from namespace.client_secret because it is an empty string", Mock(client_id="12345678123456781234567812345678", client_secret=""), RequiredArgumentMissingError),
    ("should not raise any exception when namespace.client_id is a valid input for creating a UUID and namespace.client_secret has a valid str representation", Mock(client_id="12345678123456781234567812345678", client_secret="12345"), None)
]


@pytest.mark.parametrize("test_description, namespace, expected_exception", test_validate_client_id_data, ids=[i[0] for i in test_validate_client_id_data])
def test_validate_client_id(test_description, namespace, expected_exception):
    if expected_exception is None:
        validate_client_id(namespace)
    else:
        with pytest.raises(expected_exception):
            validate_client_id(namespace)


test_validate_client_secret_data = [
    ("should not raise any exception when isCreate is false", False, Mock(client_id=None), None),
    ("should not raise any exception when namespace.client_secret is None", True, Mock(client_secret=None), None),
    ("should raise RequiredArgumentMissingError exception when namespace.client_id is None and client_secret is not None", True, Mock(client_id=None, client_secret="123"), RequiredArgumentMissingError),
    ("should raise RequiredArgumentMissingError exception when can not crate a string representation from namespace.client_id because it is empty", True, Mock(client_id="", client_secret="123"), RequiredArgumentMissingError),
    ("should raise RequiredArgumentMissingError exception when can not crate a string representation from namespace.client_id because it is None", True, Mock(client_id=None, client_secret="123"), RequiredArgumentMissingError)
]


@pytest.mark.parametrize("test_description, isCreate, namespace, expected_exception", test_validate_client_secret_data, ids=[i[0] for i in test_validate_client_secret_data])
def test_validate_client_secret(test_description, isCreate, namespace, expected_exception):
    validate_client_secret_fn = validate_client_secret(isCreate)
    if expected_exception is None:
        validate_client_secret_fn(namespace)
    else:
        with pytest.raises(expected_exception):
            validate_client_secret_fn(namespace)


test_validate_cluster_resource_group_data = [
    ("should not raise any exception when namespace.cluster_resource_group is None", None, None, Mock(cluster_resource_group=None), None),
    ("should raise InvalidArgumentValueError exception when namespace.cluster_resource_group is not None and resource group exists in the client returned by get_mgmt_service_client", Mock(**{"resource_groups.check_existence.return_value": True}), Mock(cli_ctx=None), Mock(cluster_resource_group="some_resource_group"), InvalidArgumentValueError),
    ("should not raise any exception when namespace.cluster_resource_group is not None and resource group does not exists in the client returned by get_mgmt_service_client", Mock(**{"resource_groups.check_existence.return_value": False}), Mock(cli_ctx=None), Mock(cluster_resource_group="some_resource_group"), None)
]


@pytest.mark.parametrize("test_description, client_mock, cmd_mock, namespace, expected_exception", test_validate_cluster_resource_group_data, ids=[i[0] for i in test_validate_cluster_resource_group_data])
@patch('azext_aro._validators.get_mgmt_service_client')
def test_validate_cluster_resource_group(get_mgmt_service_client_mock, test_description, client_mock, cmd_mock, namespace, expected_exception):
    get_mgmt_service_client_mock.return_value = client_mock
    if expected_exception is None:
        validate_cluster_resource_group(cmd_mock, namespace)
    else:
        with pytest.raises(expected_exception):
            validate_cluster_resource_group(cmd_mock, namespace)


test_validate_disk_encryption_set_data = [
    ("should not raise any exception when namespace.disk_encryption_set is None", None, Mock(disk_encryption_set=None), None, None, None, None),
    ("should raise InvalidArgumentValueError exception when namespace.disk_encryption_set is not None and is_valid_resource_id(namespace.disk_encryption_set) returns False", None, Mock(disk_encryption_set="something different than None"), False, None, InvalidArgumentValueError, None),
    ("should not raise any exception when compute_client.disk_encryption_sets.get() not raises CludError exception", Mock(cli_ctx=None), Mock(disk_encryption_set="something different than None"), True, Mock(), None, {"resource_group": None, "name": None})
]


@pytest.mark.parametrize("test_description, cmd_mock, namespace, is_valid_resource_id_return_value, compute_client_mock, expected_exception, parse_resource_id_mock_return_value", test_validate_disk_encryption_set_data, ids=[i[0] for i in test_validate_disk_encryption_set_data])
@patch('azext_aro._validators.get_mgmt_service_client')
@patch('azext_aro._validators.parse_resource_id')
@patch('azext_aro._validators.is_valid_resource_id')
def test_validate_disk_encryption_set(is_valid_resource_id_mock, parse_resource_id_mock, get_mgmt_service_client_mock, test_description, cmd_mock, namespace, is_valid_resource_id_return_value, compute_client_mock, expected_exception, parse_resource_id_mock_return_value):
    is_valid_resource_id_mock.return_value = is_valid_resource_id_return_value
    parse_resource_id_mock.return_value = parse_resource_id_mock_return_value

    if compute_client_mock is not None:
        compute_client_mock.get.return_value = None
        get_mgmt_service_client_mock.return_value = compute_client_mock

    if expected_exception is None:
        validate_disk_encryption_set(cmd_mock, namespace)
    else:
        with pytest.raises(expected_exception):
            validate_disk_encryption_set(cmd_mock, namespace)


test_validate_domain_data = [
    ("should not raise any exception when namespace.domain is None", Mock(domain=None), None),
    ("should not raise any exception when namespace.domain has '-'", Mock(domain="my-domain.com"), None),
    ("should not raise any exception when namespace.domain is google.com.au", Mock(domain="google.com.au"), None),
    ("should not raise any exception when namespace.domain is some.more.than.expected", Mock(domain="some.more.than.expected"), None),
    ("should not raise any exception when namespace.domain is azure.microsoft.com", Mock(domain="azure.microsoft.com"), None),
    ("should raise InvalidArgumentValueError exception when namespace.domain ends with '.'", Mock(domain="google."), InvalidArgumentValueError),
    ("should raise InvalidArgumentValueError exception when namespace.domain has '_'", Mock(domain="my_domain.com"), InvalidArgumentValueError),
    ("should raise InvalidArgumentValueError exception when namespace.domain has is google..com", Mock(domain="google..com"), InvalidArgumentValueError)
]


@pytest.mark.parametrize("test_description, namespace, expected_exception", test_validate_domain_data, ids=[i[0] for i in test_validate_domain_data])
def test_validate_domain(test_description, namespace, expected_exception):
    if expected_exception is None:
        validate_domain(namespace)
    else:
        with pytest.raises(expected_exception):
            validate_domain(namespace)


test_validate_pull_secret_data = [
    ("should not raise any exception when namespace.pull_secret is None", Mock(pull_secret=None), None),
    ("should not raise any exception when namespace.pull_secret is a valid JSON", Mock(pull_secret='{"key":"value"}'), None),
    ("should raise InvalidArgumentValueError exception when namespace.pull_secret is not a valid JSON beacuse is an empty string", Mock(pull_secret=""), InvalidArgumentValueError),
    ("should raise InvalidArgumentValueError exception when namespace.pull_secret is not a valid JSON because missing value", Mock(pull_secret='{"key": }'), InvalidArgumentValueError),
    ("should raise InvalidArgumentValueError exception when namespace.pull_secret is not a valid JSON because is a simple string", Mock(pull_secret='a simple string'), InvalidArgumentValueError)
]


@pytest.mark.parametrize("test_description, namespace, expected_exception", test_validate_pull_secret_data, ids=[i[0] for i in test_validate_pull_secret_data])
def test_validate_pull_secret(test_description, namespace, expected_exception):
    if expected_exception is None:
        validate_pull_secret(namespace)
    else:
        with pytest.raises(expected_exception):
            validate_pull_secret(namespace)


test_validate_sdn_data = [
    ("should not raise any exception when namespace.software_defined_network is None", Mock(software_defined_network=None), None),
    ("should not raise any exception when namespace.software_defined_network is OVNKubernetes", Mock(software_defined_network="OVNKubernetes"), None),
    ("should not raise any exception when namespace.software_defined_network is OpenshiftSDN", Mock(software_defined_network="OpenshiftSDN"), None),
    ("should raise InvalidArgumentValueError exception when namespace.software_defined_network is 'uknown', not one of the target values", Mock(software_defined_network="uknown"), InvalidArgumentValueError),
    ("should raise InvalidArgumentValueError exception when namespace.software_defined_network is an empty string", Mock(software_defined_network=""), InvalidArgumentValueError)
]


@pytest.mark.parametrize("test_description, namespace, expected_exception", test_validate_sdn_data, ids=[i[0] for i in test_validate_sdn_data])
def test_validate_sdn(test_description, namespace, expected_exception):
    if expected_exception is None:
        validate_sdn(namespace)
    else:
        with pytest.raises(expected_exception):
            validate_sdn(namespace)


@patch('azext_aro._validators.get_mgmt_service_client')
@patch('azext_aro._validators.get_subscription_id')
@patch('azext_aro._validators.parse_resource_id')
@patch('azext_aro._validators.is_valid_resource_id')
def test_validate_subnet(is_valid_resource_id_mock, parse_resource_id_mock, get_subscription_id_mock, get_mgmt_service_client_mock):
    class TestData():
        def __init__(self, test_description: str = None, namespace: Mock = None, key: str = None, is_valid_resource_id_mock_return_value: bool = None, parse_resource_id_mock_return_value: dict = None, get_subscription_id_mock_return_value: any = None, get_mgmt_service_client_mock_return_value: any = None, cmd: Mock = None, expected_exception: Exception = None) -> None:
            self.test_description = test_description
            self.namespace = namespace
            self.key = key
            self.is_valid_resource_id_mock_return_value = is_valid_resource_id_mock_return_value
            self.parse_resource_id_mock_return_value = parse_resource_id_mock_return_value
            self.get_subscription_id_mock_return_value = get_subscription_id_mock_return_value
            self.get_mgmt_service_client_mock_return_value = get_mgmt_service_client_mock_return_value
            self.cmd = cmd
            self.expected_exception = expected_exception

    testcases: List[TestData] = [
        TestData(
            test_description="should raise RequiredArgumentMissingError exception when subnet is not a valid resource id and namespace.vnet is False",
            namespace=Mock(key='192.168.0.0/28', vnet=False),
            key='key',
            is_valid_resource_id_mock_return_value=False,
            expected_exception=RequiredArgumentMissingError
        ),
        TestData(
            test_description="should raise InvalidArgumentValueError exception when parts subscription is different than cluster_subscription",
            namespace=Mock(key='192.168.0.0/28', vnet=False),
            key='key',
            is_valid_resource_id_mock_return_value=True,
            parse_resource_id_mock_return_value={"subscription": "expected"},
            get_subscription_id_mock_return_value="different than expected",
            cmd=Mock(cli_ctx=None),
            expected_exception=InvalidArgumentValueError
        ),
        TestData(
            test_description="should raise InvalidArgumentValueError exception when parts namespace is different than expected namespace",
            namespace=Mock(key='192.168.0.0/28', vnet=False),
            key='key',
            is_valid_resource_id_mock_return_value=True,
            parse_resource_id_mock_return_value={
                "subscription": "subscription", "namespace": "something.different"},
            get_subscription_id_mock_return_value="subscription",
            cmd=Mock(cli_ctx=None),
            expected_exception=InvalidArgumentValueError
        ),
        TestData(
            test_description="should raise InvalidArgumentValueError exception when parts type is different than expected_parts",
            namespace=Mock(key='192.168.0.0/28', vnet=False),
            key='key',
            is_valid_resource_id_mock_return_value=True,
            parse_resource_id_mock_return_value={
                "subscription": "subscription", "namespace": "MICROSOFT.NETWORK", "type": "this_should_be_virtualnetworks"},
            get_subscription_id_mock_return_value="subscription",
            cmd=Mock(cli_ctx=None),
            expected_exception=InvalidArgumentValueError
        ),
        TestData(
            test_description="should raise InvalidArgumentValueError exception when subnet childs is different than expected childs",
            namespace=Mock(key='192.168.0.0/28', vnet=False),
            key='key',
            is_valid_resource_id_mock_return_value=True,
            parse_resource_id_mock_return_value={
                "subscription": "subscription", "namespace": "MICROSOFT.NETWORK", "type": "virtualnetworks", "last_child_num": 0},
            get_subscription_id_mock_return_value="subscription",
            cmd=Mock(cli_ctx=None),
            expected_exception=InvalidArgumentValueError
        ),
        TestData(
            test_description="should raise InvalidArgumentValueError exception when subnet has child namespace",
            namespace=Mock(key='192.168.0.0/28', vnet=False),
            key='key',
            is_valid_resource_id_mock_return_value=True,
            parse_resource_id_mock_return_value={"subscription": "subscription", "namespace": "MICROSOFT.NETWORK",
                                                 "type": "virtualnetworks", "last_child_num": 1, "child_namespace_1": "something"},
            get_subscription_id_mock_return_value="subscription",
            cmd=Mock(cli_ctx=None),
            expected_exception=InvalidArgumentValueError
        ),
        TestData(
            test_description="should raise InvalidArgumentValueError exception when child type subnet do not equal subnets",
            namespace=Mock(key='192.168.0.0/28', vnet=False),
            key='key',
            is_valid_resource_id_mock_return_value=True,
            parse_resource_id_mock_return_value={"subscription": "subscription", "namespace": "MICROSOFT.NETWORK",
                                                 "type": "virtualnetworks", "last_child_num": 1, "child_type_1": "this_should_be_subnets"},
            get_subscription_id_mock_return_value="subscription",
            cmd=Mock(cli_ctx=None),
            expected_exception=InvalidArgumentValueError
        ),
        TestData(
            test_description="should raise InvalidArgumentValueError exception when client.subnets.get raises CLIInternalError because client.subnets.get() raises Exception exception",
            namespace=Mock(key='192.168.0.0/28', vnet=False),
            key='key',
            is_valid_resource_id_mock_return_value=True,
            parse_resource_id_mock_return_value={"subscription": "subscription", "namespace": "MICROSOFT.NETWORK",
                                                 "type": "virtualnetworks", "last_child_num": 1, "child_type_1": "subnets"},
            get_subscription_id_mock_return_value="subscription",
            get_mgmt_service_client_mock_return_value=Mock(
                **{"subnets.get.side_effect": Exception}),
            cmd=Mock(cli_ctx=None),
            expected_exception=CLIInternalError
        ),
        TestData(
            test_description="should raise InvalidArgumentValueError exception when client.subnets.get() raises ResourceNotFoundError exception",
            namespace=Mock(key='192.168.0.0/28', vnet=False),
            key='key',
            is_valid_resource_id_mock_return_value=True,
            parse_resource_id_mock_return_value={"subscription": "subscription", "namespace": "MICROSOFT.NETWORK", "type": "virtualnetworks",
                                                 "last_child_num": 1, "child_type_1": "subnets", "resource_group": None, "name": None, "child_name_1": None},
            get_subscription_id_mock_return_value="subscription",
            get_mgmt_service_client_mock_return_value=Mock(
                **{"subnets.get.side_effect": ResourceNotFoundError("")}),
            cmd=Mock(cli_ctx=None),
            expected_exception=InvalidArgumentValueError
        ),
        TestData(
            test_description="should not raise any exception",
            namespace=Mock(key='192.168.0.0/28', vnet=False),
            key='key',
            is_valid_resource_id_mock_return_value=True,
            parse_resource_id_mock_return_value={"subscription": "subscription", "namespace": "MICROSOFT.NETWORK", "type": "virtualnetworks",
                                                 "last_child_num": 1, "child_type_1": "subnets", "resource_group": None, "name": None, "child_name_1": None},
            get_subscription_id_mock_return_value="subscription",
            get_mgmt_service_client_mock_return_value=Mock(
                **{"subnets.get.return_value": None}),
            cmd=Mock(cli_ctx=None)
        )
    ]

    for tc in testcases:
        is_valid_resource_id_mock.return_value = tc.is_valid_resource_id_mock_return_value
        parse_resource_id_mock.return_value = tc.parse_resource_id_mock_return_value
        get_subscription_id_mock.return_value = tc.get_subscription_id_mock_return_value
        get_mgmt_service_client_mock.return_value = tc.get_mgmt_service_client_mock_return_value

        validate_subnet_fn = validate_subnet(tc.key)

        if tc.expected_exception is None:
            validate_subnet_fn(tc.cmd, tc.namespace)
        else:
            with pytest.raises(tc.expected_exception):
                validate_subnet_fn(tc.cmd, tc.namespace)


@patch('azext_aro._validators.parse_resource_id')
def test_validate_subnets(parse_resource_id_mock):
    class TestData():
        def __init__(self, test_description: str = None, parse_resource_id_mock: any = None, expected_exception: Exception = None) -> None:
            self.test_description = test_description
            self.parse_resource_id_mock = parse_resource_id_mock
            self.expected_exception = expected_exception

    testcases: List[TestData] = [
        TestData(
            test_description="should raise InvalidArgumentValueError exception when resource group of master_parts is different than resource_group of worker_parts",
            parse_resource_id_mock=[
                {"resource_group": "a"}, {"resource_group": "b"}],
            expected_exception=InvalidArgumentValueError
        ),
        TestData(
            test_description="should raise InvalidArgumentValueError exception when name of master_parts is different than name of worker_parts",
            parse_resource_id_mock=[{"resource_group": "equal", "name": "something"}, {
                "resource_group": "equal", "name": "different"}],
            expected_exception=InvalidArgumentValueError
        ),
        TestData(
            test_description="should raise InvalidArgumentValueError exception when name of master_parts is different than name of worker_parts",
            parse_resource_id_mock=[{"resource_group": "equal", "name": "something", "child_name_1": "should_not_be_equal"},
                                    {"resource_group": "equal", "name": "something", "child_name_1": "should_not_be_equal"}],
            expected_exception=InvalidArgumentValueError
        ),
        TestData(
            test_description="should not raise any exception",
            parse_resource_id_mock=[{"resource_group": "equal", "name": "something", "child_name_1": "a"},
                                    {"resource_group": "equal", "name": "something", "child_name_1": "z"}]
        )
    ]

    for tc in testcases:
        parse_resource_id_mock.side_effect = tc.parse_resource_id_mock

        if tc.expected_exception is None:
            validate_subnets(None, None)
        else:
            with pytest.raises(tc.expected_exception):
                validate_subnets(None, None)


def test_validate_visibility():
    class TestData():
        def __init__(self, test_description: str = None, key: Mock = None, namespace: Mock = None, expected_exception: Exception = None) -> None:
            self.test_description = test_description
            self.key = key
            self.namespace = namespace
            self.expected_exception = expected_exception

    testcases: List[TestData] = [
        TestData(
            test_description="should not raise any exception",
            key="key",
            namespace=Mock(key=None)
        ),
        TestData(
            test_description="should raise InvalidArgumentValueError exception because visibility is not one of the expected values",
            key="key",
            namespace=Mock(key="super_private"),
            expected_exception=InvalidArgumentValueError
        ),
        TestData(
            test_description="should not raise any exception because visibility is private",
            key="key",
            namespace=Mock(key="private")
        ),
        TestData(
            test_description="should not raise any exception because visibility is PRIVATE",
            key="key",
            namespace=Mock(key="PRIVATE"),
        ),
        TestData(
            test_description="should not raise any exception because visibility is PUBLIC",
            key="key",
            namespace=Mock(key="PUBLIC"),
        ),
        TestData(
            test_description="should not raise any exception because visibility is public",
            key="key",
            namespace=Mock(key="public")
        )
    ]

    for tc in testcases:
        validate_visibility_fn = validate_visibility(tc.key)

        if tc.expected_exception is None:
            validate_visibility_fn(tc.namespace)
        else:
            with pytest.raises(tc.expected_exception):
                validate_visibility_fn(tc.namespace)


def test_validate_vnet_resource_group_name():
    class TestData():
        def __init__(self, test_description: str = None, namespace: Mock = None, expected_namespace_vnet_resource_group_name: str = None) -> None:
            self.test_description = test_description
            self.namespace = namespace
            self.expected_namespace_vnet_resource_group_name = expected_namespace_vnet_resource_group_name

    testcases: List[TestData] = [
        TestData(
            test_description="should not copy namespace.resource_group_name to namespace.vnet_resource_group_name because namespace.resource_group_name already has a value",
            namespace=Mock(vnet_resource_group_name="hello",
                           resource_group_name="this_will_not_be_copied"),
            expected_namespace_vnet_resource_group_name="hello"
        ),
        TestData(
            test_description="should copy resource_group_name field to vnet_resource_group_name because namespace.vnet_resource_group_name is None",
            namespace=Mock(vnet_resource_group_name=None,
                           resource_group_name="will_copy_this"),
            expected_namespace_vnet_resource_group_name="will_copy_this"
        )
    ]

    for tc in testcases:
        validate_vnet_resource_group_name(tc.namespace)

        assert(tc.namespace.vnet_resource_group_name ==
               tc.expected_namespace_vnet_resource_group_name)


def test_validate_worker_count():
    class TestData():
        def __init__(self, test_description: str = None, namespace: Mock = None, expected_exception: Exception = None) -> None:
            self.test_description = test_description
            self.namespace = namespace
            self.expected_exception = expected_exception

    testcases: List[TestData] = [
        TestData(
            test_description="should not raise any Exception because worker count of namespace is None",
            namespace=Mock(worker_count=None)
        ),
        TestData(
            test_description="should not raise any Exception because worker count of namespace is 3",
            namespace=Mock(worker_count=None)
        ),
        TestData(
            test_description="should raise InvalidArgumentValueError Exception because worker count of namespace is less than minimum workers count",
            namespace=Mock(worker_count=2),
            expected_exception=InvalidArgumentValueError
        )
    ]

    for tc in testcases:
        if tc.expected_exception is None:
            validate_worker_count(tc.namespace)
        else:
            with pytest.raises(tc.expected_exception):
                validate_worker_count(tc.namespace)


def test_validate_worker_vm_disk_size_gb():
    class TestData():
        def __init__(self, test_description: str = None, namespace: Mock = None, expected_exception: Exception = None) -> None:
            self.test_description = test_description
            self.namespace = namespace
            self.expected_exception = expected_exception

    testcases: List[TestData] = [
        TestData(
            test_description="should not raise any Exception because worker_vm_disk_size_gb of namespace is None",
            namespace=Mock(worker_vm_disk_size_gb=None)
        ),
        TestData(
            test_description="should raise InvalidArgumentValueError Exception because worker_vm_disk_size_gb of namespace is less than minimum_worker_vm_disk_size_gb",
            namespace=Mock(worker_vm_disk_size_gb=2),
            expected_exception=InvalidArgumentValueError
        ),
        TestData(
            test_description="should not raise any Exception because worker_vm_disk_size_gb of namespace is equal than minimum_worker_vm_disk_size_gb",
            namespace=Mock(worker_vm_disk_size_gb=128),
            expected_exception=None
        ),
        TestData(
            test_description="should not raise any Exception because worker_vm_disk_size_gb of namespace is greater than minimum_worker_vm_disk_size_gb",
            namespace=Mock(worker_vm_disk_size_gb=220),
            expected_exception=None
        ),
    ]

    for tc in testcases:
        if tc.expected_exception is None:
            validate_worker_vm_disk_size_gb(tc.namespace)
        else:
            with pytest.raises(tc.expected_exception):
                validate_worker_vm_disk_size_gb(tc.namespace)


def test_validate_refresh_cluster_credentials():
    class TestData():
        def __init__(self, test_description: str = None, namespace: Mock = None, expected_exception: Exception = None) -> None:
            self.test_description = test_description
            self.namespace = namespace
            self.expected_exception = expected_exception

    testcases: List[TestData] = [
        TestData(
            test_description="should not raise any Exception because namespace.refresh_cluster_credentials is none",
            namespace=Mock(refresh_cluster_credentials=None)
        ),
        TestData(
            test_description="should raise RequiredArgumentMissingError Exception because namespace.client_secret is not None",
            namespace=Mock(client_secret="secret_123"),
            expected_exception=RequiredArgumentMissingError
        ),
        TestData(
            test_description="should raise RequiredArgumentMissingError Exception because namespace.client_id is not None",
            namespace=Mock(client_id="client_id_456"),
            expected_exception=RequiredArgumentMissingError
        ),
        TestData(
            test_description="should not raise any Exception because namespace.client_secret is None and namespace.client_id is None",
            namespace=Mock(client_secret=None, client_id=None)
        )
    ]

    for tc in testcases:
        if tc.expected_exception is None:
            validate_refresh_cluster_credentials(tc.namespace)
        else:
            with pytest.raises(tc.expected_exception):
                validate_refresh_cluster_credentials(tc.namespace)
