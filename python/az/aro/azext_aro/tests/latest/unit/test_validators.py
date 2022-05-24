# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

from typing import Dict, List
from unittest import TestCase
from unittest.mock import Mock, patch
from azext_aro._validators import validate_cidr, validate_client_id, validate_client_secret, validate_cluster_resource_group, validate_disk_encryption_set, validate_domain, validate_pull_secret, validate_sdn, validate_subnet
from azure.cli.core.azclierror import InvalidArgumentValueError, RequiredArgumentMissingError, RequiredArgumentMissingError, CLIInternalError
from azure.core.exceptions import ResourceNotFoundError


class TestValidators(TestCase):
    def test_validate_cidr(self):
        class TestData():
            def __init__(self, test_description: str = None, dummyclass: Mock = None, attribute_to_get_from_object: str = None, expected_exception: Exception = None) -> None:
                self.test_description = test_description
                self.dummyclass = dummyclass
                self.attribute_to_get_from_object = attribute_to_get_from_object
                self.expected_exception = expected_exception

        testcases: List[TestData] = [
            TestData(
                test_description="should not raise exception when valid IPv4 address",
                dummyclass=Mock(key='192.168.0.0/28'),
                attribute_to_get_from_object='key',
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError when non valid IPv4 address due to beeing a simple string",
                dummyclass=Mock(key='this is an invalid network'),
                attribute_to_get_from_object='key',
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError when non valid IPv4 address due to invalid network ID",
                dummyclass=Mock(key='192.168.0.0.0.0/28'),
                attribute_to_get_from_object='key',
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError when non valid IPv4 address due to invalid hostID",
                dummyclass=Mock(key='192.168.0.0.0.0/2888'),
                attribute_to_get_from_object='key',
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should not raise exception when IPv4 address is None ",
                dummyclass=Mock(key=None),
                attribute_to_get_from_object='key'
            )
        ]

        for tc in testcases:
            validate_cidr_fn = validate_cidr(tc.attribute_to_get_from_object)
            if tc.expected_exception is None:
                validate_cidr_fn(tc.dummyclass)
            else:
                with self.assertRaises(tc.expected_exception, msg=tc.test_description):
                    validate_cidr_fn(tc.dummyclass)

    def test_validate_client_id(self):
        class TestData():
            def __init__(self, test_description: str = None, namespace: Mock = None, expected_exception: Exception = None) -> None:
                self.test_description = test_description
                self.namespace = namespace
                self.expected_exception = expected_exception

        testcases: List[TestData] = [
            TestData(
                test_description="should return None when namespace.client_id is None",
                namespace=Mock(client_id=None)
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError when it can not create a UUID from namespace.client_id",
                namespace=Mock(client_id="invalid_client_id"),
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should raise RequiredArgumentMissingError when can not crate a string representation from namespace.client_secret because is None",
                namespace=Mock(client_id="12345678123456781234567812345678", client_secret=None),
                expected_exception=RequiredArgumentMissingError
            ),
            TestData(
                test_description="should raise RequiredArgumentMissingError when can not crate a string representation from namespace.client_secret because it is an empty string",
                namespace=Mock(client_id="12345678123456781234567812345678", client_secret=""),
                expected_exception=RequiredArgumentMissingError
            ),
            TestData(
                test_description="should not raise exception when namespace.client_id is a valid input for creating a UUID and namespace.client_secret has a valid str representation",
                namespace=Mock(client_id="12345678123456781234567812345678", client_secret="12345")
            )
        ]

        for tc in testcases:
            if tc.expected_exception is None:
                validate_client_id(tc.namespace)
            else:
                with self.assertRaises(tc.expected_exception, msg=tc.test_description):
                    validate_client_id(tc.namespace)

    def test_validate_client_secret(self):
        class TestData():
            def __init__(self, test_description: str = None, isCreate: bool = None, namespace: Mock = None, expected_exception: Exception = None) -> None:
                self.test_description = test_description
                self.isCreate = isCreate
                self.namespace = namespace
                self.expected_exception = expected_exception

        testcases: List[TestData] = [
            TestData(
                test_description="should not raise exception when isCreate is false",
                isCreate=False,
                namespace=Mock(client_id=None)
            ),
            TestData(
                test_description="should not raise exception when namespace.client_secret is None",
                isCreate=True,
                namespace=Mock(client_secret=None)
            ),
            TestData(
                test_description="should raise RequiredArgumentMissingError exception when namespace.client_id is None and client_secret is not None",
                isCreate=True,
                namespace=Mock(client_id=None, client_secret="123"),
                expected_exception=RequiredArgumentMissingError
            ),
            TestData(
                test_description="should raise RequiredArgumentMissingError exception when can not crate a string representation from namespace.client_id because it is empty",
                isCreate=True,
                namespace=Mock(client_id="", client_secret="123"),
                expected_exception=RequiredArgumentMissingError
            ),
            TestData(
                test_description="should raise RequiredArgumentMissingError exception when can not crate a string representation from namespace.client_id because it is None",
                isCreate=True,
                namespace=Mock(client_id=None, client_secret="123"),
                expected_exception=RequiredArgumentMissingError
            )
        ]

        for tc in testcases:
            validate_client_secret_fn = validate_client_secret(tc.isCreate)
            if tc.expected_exception is None:
                validate_client_secret_fn(tc.namespace)
            else:
                with self.assertRaises(tc.expected_exception, msg=tc.test_description):
                    validate_client_secret_fn(tc.namespace)

    @patch('azext_aro._validators.get_mgmt_service_client')
    def test_validate_cluster_resource_group(self, get_mgmt_service_client_mock):
        class TestData():
            def __init__(self, test_description: str = None, client_mock: Mock = None, cmd_mock: Mock = None, namespace: Mock = None, expected_exception: Exception = None) -> None:
                self.test_description = test_description
                self.client_mock = client_mock
                self.cmd_mock = cmd_mock
                self.namespace = namespace
                self.expected_exception = expected_exception

        testcases: List[TestData] = [
            TestData(
                test_description="should not raise any exception when namespace.cluster_resource_group is None",
                namespace=Mock(cluster_resource_group=None)
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when namespace.cluster_resource_group is not None and resource group exists in the client returned by get_mgmt_service_client",
                client_mock=Mock(**{"resource_groups.check_existence.return_value": True}),
                cmd_mock=Mock(cli_ctx=None),
                namespace=Mock(cluster_resource_group="some_resource_group"),
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should not raise any exception when namespace.cluster_resource_group is not None and resource group does not exists in the client returned by get_mgmt_service_client",
                client_mock=Mock(**{"resource_groups.check_existence.return_value": False}),
                cmd_mock=Mock(cli_ctx=None),
                namespace=Mock(cluster_resource_group="some_resource_group")
            ),
        ]

        for tc in testcases:
            get_mgmt_service_client_mock.return_value = tc.client_mock

            if tc.expected_exception is None:
                validate_cluster_resource_group(tc.cmd_mock, tc.namespace)
            else:
                with self.assertRaises(tc.expected_exception, msg=tc.test_description):
                    validate_cluster_resource_group(tc.cmd_mock, tc.namespace)

    @patch('azext_aro._validators.get_mgmt_service_client')
    @patch('azext_aro._validators.parse_resource_id')
    @patch('azext_aro._validators.is_valid_resource_id')
    def test_validate_disk_encryption_set(self, is_valid_resource_id_mock, parse_resource_id_mock, get_mgmt_service_client_mock):
        class TestData():
            def __init__(self, test_description: str = None, cmd_mock: Mock = None, namespace: Mock = None, is_valid_resource_id_return_value: bool = None, compute_client_mock: Mock = None, expected_exception: Exception = None, parse_resource_id_mock_return_value: Dict = None) -> None:
                self.test_description = test_description
                self.cmd_mock = cmd_mock
                self.namespace = namespace
                self.is_valid_resource_id_return_value = is_valid_resource_id_return_value
                self.compute_client_mock = compute_client_mock
                self.expected_exception = expected_exception
                self.parse_resource_id_mock_return_value = parse_resource_id_mock_return_value

        testcases: List[TestData] = [
            TestData(
                test_description="should not raise any exception when namespace.disk_encryption_set is None",
                namespace=Mock(disk_encryption_set=None)
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when namespace.disk_encryption_set is not None and is_valid_resource_id(namespace.disk_encryption_set) returns False",
                namespace=Mock(disk_encryption_set="something different than None"),
                is_valid_resource_id_return_value=False,
                expected_exception=InvalidArgumentValueError,
            ),
            TestData(
                test_description="should not raise any exception when compute_client.disk_encryption_sets.get() not raises CludError exception",
                cmd_mock=Mock(cli_ctx=None),
                namespace=Mock(disk_encryption_set="something different than None"),
                is_valid_resource_id_return_value=True,
                compute_client_mock=Mock(),
                parse_resource_id_mock_return_value={"resource_group": None, "name": None}
            )
        ]
        for tc in testcases:
            is_valid_resource_id_mock.return_value = tc.is_valid_resource_id_return_value
            parse_resource_id_mock.return_value = tc.parse_resource_id_mock_return_value

            if tc.compute_client_mock is not None:
                tc.compute_client_mock.get.return_value = None
                get_mgmt_service_client_mock.return_value = tc.compute_client_mock

            if tc.expected_exception is None:
                validate_disk_encryption_set(tc.cmd_mock, tc.namespace)
            else:
                with self.assertRaises(tc.expected_exception, msg=tc.test_description):
                    validate_disk_encryption_set(tc.cmd_mock, tc.namespace)

    def test_validate_domain(self):
        class TestData():
            def __init__(self, test_description: str = None, namespace: Mock = None, expected_exception: Exception = None) -> None:
                self.test_description = test_description
                self.namespace = namespace
                self.expected_exception = expected_exception

        testcases: List[TestData] = [
            TestData(
                test_description="should not raise any exception when namespace.domain is None",
                namespace=Mock(domain=None)
            ),
            TestData(
                test_description="should not raise any exception when namespace.domain has '-'",
                namespace=Mock(domain="my-domain.com")
            ),
            TestData(
                test_description="should not raise any exception when namespace.domain is google.com.au",
                namespace=Mock(domain="google.com.au")
            ),
            TestData(
                test_description="should not raise any exception when namespace.domain is some.more.than.expected",
                namespace=Mock(domain="some.more.than.expected")
            ),
            TestData(
                test_description="should not raise any exception when namespace.domain is azure.microsoft.com",
                namespace=Mock(domain="azure.microsoft.com")
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when namespace.domain ends with '.'",
                namespace=Mock(domain="google."),
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when namespace.domain has '_'",
                namespace=Mock(domain="my_domain.com"),
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when namespace.domain has is google..com",
                namespace=Mock(domain="google..com"),
                expected_exception=InvalidArgumentValueError
            )
        ]

        for tc in testcases:
            if tc.expected_exception is None:
                validate_domain(tc.namespace)
            else:
                with self.assertRaises(tc.expected_exception, msg=tc.test_description):
                    validate_domain(tc.namespace)

    def test_validate_pull_secret(self):
        class TestData():
            def __init__(self, test_description: str = None, namespace: Mock = None, expected_exception: Exception = None) -> None:
                self.test_description = test_description
                self.namespace = namespace
                self.expected_exception = expected_exception

        testcases: List[TestData] = [
            TestData(
                test_description="should not raise any exception when namespace.pull_secret is None",
                namespace=Mock(pull_secret=None)
            ),
            TestData(
                test_description="should not raise any exception when namespace.pull_secret is a valid JSON",
                namespace=Mock(pull_secret='{"key":"value"}')
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when namespace.pull_secret is not a valid JSON beacuse is an empty string",
                namespace=Mock(pull_secret=""),
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when namespace.pull_secret is not a valid JSON because missing value",
                namespace=Mock(pull_secret='{"key": }'),
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when namespace.pull_secret is not a valid JSON because is a simple string",
                namespace=Mock(pull_secret='a simple string'),
                expected_exception=InvalidArgumentValueError
            )
        ]

        for tc in testcases:
            if tc.expected_exception is None:
                validate_pull_secret(tc.namespace)
            else:
                with self.assertRaises(tc.expected_exception, msg=tc.test_description):
                    validate_pull_secret(tc.namespace)

    def test_validate_sdn(self):
        class TestData():
            def __init__(self, test_description: str = None, namespace: Mock = None, expected_exception: Exception = None) -> None:
                self.test_description = test_description
                self.namespace = namespace
                self.expected_exception = expected_exception

        testcases: List[TestData] = [
            TestData(
                test_description="should not raise any exception when namespace.software_defined_network is None",
                namespace=Mock(software_defined_network=None)
            ),
            TestData(
                test_description="should not raise any exception when namespace.software_defined_network is OVNKubernetes",
                namespace=Mock(software_defined_network="OVNKubernetes")
            ),
            TestData(
                test_description="should not raise any exception when namespace.software_defined_network is OpenshiftSDN",
                namespace=Mock(software_defined_network="OpenshiftSDN")
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when namespace.software_defined_network is 'uknown', not one of the target values",
                namespace=Mock(software_defined_network="uknown"),
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when namespace.software_defined_network is an empty string",
                namespace=Mock(software_defined_network=""),
                expected_exception=InvalidArgumentValueError
            )
        ]

        for tc in testcases:
            if tc.expected_exception is None:
                validate_sdn(tc.namespace)
            else:
                with self.assertRaises(tc.expected_exception, msg=tc.test_description):
                    validate_sdn(tc.namespace)

    @patch('azext_aro._validators.get_mgmt_service_client')
    @patch('azext_aro._validators.get_subscription_id')
    @patch('azext_aro._validators.parse_resource_id')
    @patch('azext_aro._validators.is_valid_resource_id')
    def test_validate_subnet(self, is_valid_resource_id_mock, parse_resource_id_mock, get_subscription_id_mock, get_mgmt_service_client_mock):
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
                parse_resource_id_mock_return_value={"subscription": "subscription", "namespace": "something.different"},
                get_subscription_id_mock_return_value="subscription",
                cmd=Mock(cli_ctx=None),
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when parts type is different than expected_parts",
                namespace=Mock(key='192.168.0.0/28', vnet=False),
                key='key',
                is_valid_resource_id_mock_return_value=True,
                parse_resource_id_mock_return_value={"subscription": "subscription", "namespace": "MICROSOFT.NETWORK", "type": "this_should_be_virtualnetworks"},
                get_subscription_id_mock_return_value="subscription",
                cmd=Mock(cli_ctx=None),
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when subnet childs is different than expected childs",
                namespace=Mock(key='192.168.0.0/28', vnet=False),
                key='key',
                is_valid_resource_id_mock_return_value=True,
                parse_resource_id_mock_return_value={"subscription": "subscription", "namespace": "MICROSOFT.NETWORK", "type": "virtualnetworks", "last_child_num": 0},
                get_subscription_id_mock_return_value="subscription",
                cmd=Mock(cli_ctx=None),
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when subnet has child namespace",
                namespace=Mock(key='192.168.0.0/28', vnet=False),
                key='key',
                is_valid_resource_id_mock_return_value=True,
                parse_resource_id_mock_return_value={"subscription": "subscription", "namespace": "MICROSOFT.NETWORK", "type": "virtualnetworks", "last_child_num": 1, "child_namespace_1": "something"},
                get_subscription_id_mock_return_value="subscription",
                cmd=Mock(cli_ctx=None),
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when child type subnet do not equal subnets",
                namespace=Mock(key='192.168.0.0/28', vnet=False),
                key='key',
                is_valid_resource_id_mock_return_value=True,
                parse_resource_id_mock_return_value={"subscription": "subscription", "namespace": "MICROSOFT.NETWORK", "type": "virtualnetworks", "last_child_num": 1, "child_type_1": "this_should_be_subnets"},
                get_subscription_id_mock_return_value="subscription",
                cmd=Mock(cli_ctx=None),
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when client.subnets.get raises CLIInternalError because client.subnets.get raises Exception exception",
                namespace=Mock(key='192.168.0.0/28', vnet=False),
                key='key',
                is_valid_resource_id_mock_return_value=True,
                parse_resource_id_mock_return_value={"subscription": "subscription", "namespace": "MICROSOFT.NETWORK", "type": "virtualnetworks", "last_child_num": 1, "child_type_1": "subnets"},
                get_subscription_id_mock_return_value="subscription",
                get_mgmt_service_client_mock_return_value=Mock(**{"subnets.get.side_effect": Exception}),
                cmd=Mock(cli_ctx=None),
                expected_exception=CLIInternalError
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when client.subnets.get raises ResourceNotFoundError exception",
                namespace=Mock(key='192.168.0.0/28', vnet=False),
                key='key',
                is_valid_resource_id_mock_return_value=True,
                parse_resource_id_mock_return_value={"subscription": "subscription", "namespace": "MICROSOFT.NETWORK", "type": "virtualnetworks", "last_child_num": 1, "child_type_1": "subnets", "resource_group": None, "name": None, "child_name_1": None},
                get_subscription_id_mock_return_value="subscription",
                get_mgmt_service_client_mock_return_value=Mock(**{"subnets.get.side_effect": ResourceNotFoundError("")}),
                cmd=Mock(cli_ctx=None),
                expected_exception=InvalidArgumentValueError
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
                with self.assertRaises(tc.expected_exception, msg=tc.test_description):
                    validate_subnet_fn(tc.cmd, tc.namespace)
