# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

from collections import namedtuple
from typing import List
import unittest
from unittest.mock import Mock, patch
from azext_aro._validators import validate_cidr, validate_client_id, validate_client_secret, validate_cluster_resource_group
from azure.cli.core.azclierror import InvalidArgumentValueError, InvalidArgumentValueError, RequiredArgumentMissingError


class Namespace:
    def __init__(self, client_id=None, client_secret=None, cluster_resource_group=None):
        self.client_id = client_id
        self.client_secret = client_secret
        self.cluster_resource_group = cluster_resource_group


class TestValidators(unittest.TestCase):

    def test_validate_cidr(self):
        class Dummyclass:
            def __init__(self, key=None):
                self.key = key

        namedtuple_name = 'Testdata'
        namedtuple_attributes = ["test_description", 'dummyclass', 'attribute_to_get_from_object', "expected_exception"]
        TestData = namedtuple(namedtuple_name, namedtuple_attributes)

        testcases: List[namedtuple] = [
            TestData(
                test_description="should not raise exception when valid IPv4 address",
                dummyclass=Dummyclass('192.168.0.0/28'),
                attribute_to_get_from_object='key',
                expected_exception=None
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError when non valid IPv4 address due to beeing a simple string",
                dummyclass=Dummyclass('this is an invalid network'),
                attribute_to_get_from_object='key',
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError when non valid IPv4 address due to invalid network ID",
                dummyclass=Dummyclass('192.168.0.0.0.0/28'),
                attribute_to_get_from_object='key',
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError when non valid IPv4 address due to invalid hostID",
                dummyclass=Dummyclass('192.168.0.0.0.0/2888'),
                attribute_to_get_from_object='key',
                expected_exception=InvalidArgumentValueError
            ),
            TestData(
                test_description="should not raise exception when IPv4 address is None ",
                dummyclass=Dummyclass(None),
                attribute_to_get_from_object='key',
                expected_exception=None
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
        namedtuple_name = 'Testdata'
        namedtuple_attributes = ["test_description", 'namespace', "expected_exception"]
        TestData = namedtuple(namedtuple_name, namedtuple_attributes)

        testcases: List[namedtuple] = [
            TestData(
                "should return None when namespace.client_id is None",
                Namespace(client_id=None),
                None
            ),
            TestData(
                "should raise InvalidArgumentValueError when it can not create a UUID from namespace.client_id",
                Namespace(client_id="invalid_client_id"),
                InvalidArgumentValueError
            ),
            TestData(
                "should raise RequiredArgumentMissingError when can not crate a string representation from namespace.client_secret because is None",
                Namespace(client_id="12345678123456781234567812345678", client_secret=None),
                RequiredArgumentMissingError
            ),
            TestData(
                "should raise RequiredArgumentMissingError when can not crate a string representation from namespace.client_secret because it is an empty string",
                Namespace(client_id="12345678123456781234567812345678", client_secret=""),
                RequiredArgumentMissingError
            ),
            TestData(
                "should not raise exception when namespace.client_id is a valid input for creating a UUID and namespace.client_secret has a valid str representation",
                Namespace(client_id="12345678123456781234567812345678", client_secret="12345"),
                None
            )
        ]

        for tc in testcases:
            if tc.expected_exception is None:
                validate_client_id(tc.namespace)
            else:
                with self.assertRaises(tc.expected_exception, msg=tc.test_description):
                    validate_client_id(tc.namespace)

    def test_validate_client_secret(self):
        namedtuple_name = 'Testdata'
        namedtuple_attributes = ["test_description", "isCreate", 'namespace', "expected_exception"]
        TestData = namedtuple(namedtuple_name, namedtuple_attributes)

        testcases: List[namedtuple] = [
            TestData(
                test_description="should not raise exception when isCreate is false",
                isCreate=False,
                namespace=Namespace(client_id=None),
                expected_exception=None
            ),
            TestData(
                test_description="should not raise exception when namespace.client_secret is None",
                isCreate=True,
                namespace=Namespace(client_secret=None),
                expected_exception=None
            ),
            TestData(
                test_description="should raise RequiredArgumentMissingError exception when namespace.client_id is None and client_secret is not None",
                isCreate=True,
                namespace=Namespace(client_id=None, client_secret="123"),
                expected_exception=RequiredArgumentMissingError
            ),
            TestData(
                test_description="should raise RequiredArgumentMissingError exception when can not crate a string representation from namespace.client_id because it is empty",
                isCreate=True,
                namespace=Namespace(client_id="", client_secret="123"),
                expected_exception=RequiredArgumentMissingError
            ),
            TestData(
                test_description="should raise RequiredArgumentMissingError exception when can not crate a string representation from namespace.client_id because it is None",
                isCreate=True,
                namespace=Namespace(client_id=None, client_secret="123"),
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
        namedtuple_name = 'Testdata'
        namedtuple_attributes = ["test_description", "client_mock", "cmd_mock", 'namespace', "expected_exception"]
        TestData = namedtuple(namedtuple_name, namedtuple_attributes)

        client_mock = Mock(name="client_mock")
        client_mock.resource_groups = Mock()
        client_mock.resource_groups.check_existence.return_value = True

        cmd_mock = Mock(name="resource_groups_mock")
        cmd_mock.cli_ctx = 1

        testcases: List[namedtuple] = [
            TestData(
                test_description="should not raise exception when namespace.cluster_resource_group is None",
                client_mock=None,
                cmd_mock=None,
                namespace=Namespace(cluster_resource_group=None),
                expected_exception=None
            ),
            TestData(
                test_description="should raise InvalidArgumentValueError exception when resource group exists in the given CLI context of the client_mock",
                client_mock=client_mock,
                cmd_mock=cmd_mock,
                namespace=Namespace(cluster_resource_group="some_resource_group"),
                expected_exception=InvalidArgumentValueError
            ),
        ]

        for tc in testcases:
            get_mgmt_service_client_mock.return_value = tc.client_mock
            # self.assertEqual(get_mgmt_service_client_mock(None, None).resource_groups.check_existence(None), True)

            if tc.expected_exception is None:
                validate_cluster_resource_group(tc.cmd_mock, tc.namespace)
            else:
                #self.assertEqual(get_mgmt_service_client_mock(None, None), True)
                with self.assertRaises(tc.expected_exception, msg=tc.test_description):
                    validate_cluster_resource_group(tc.cmd_mock, tc.namespace)
