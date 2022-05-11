# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

from collections import namedtuple
from typing import List
import unittest
from azext_aro._validators import validate_cidr
from azure.cli.core.azclierror import InvalidArgumentValueError


class Dummyclass:
    def __init__(self, key=None):
        self.key = key


class TestValidators(unittest.TestCase):
    def test_validate_cidr(self):
        namedtuple_name = 'Testdata'
        namedtuple_attributes = ["test_description", 'dummyclass', 'key', "expected_exception"]
        TestData = namedtuple(namedtuple_name, namedtuple_attributes)

        testcases: List[namedtuple] = [
            TestData(
                "should not raise exception when valid IPv4Network",
                Dummyclass('192.168.0.0/28'),
                'key',
                None
            ),
            TestData(
                "should raise InvalidArgumentValueError when non valid IPv4Network due to beeing a simple string",
                Dummyclass('this is an invalid network'),
                'key',
                InvalidArgumentValueError
            ),
            TestData(
                "should raise InvalidArgumentValueError when non valid IPv4Network due to invalid network ID",
                Dummyclass('192.168.0.0.0.0/28'),
                'key',
                InvalidArgumentValueError
            ),
            TestData(
                "should raise InvalidArgumentValueError when non valid IPv4Network due to invalid hostID",
                Dummyclass('192.168.0.0.0.0/2888'),
                'key',
                InvalidArgumentValueError
            )
        ]

        for tc in testcases:
            validate_cidr_fn = validate_cidr(tc.key)
            if tc.expected_exception is None:
                result = validate_cidr_fn(tc.dummyclass)
                self.assertIsNone(result)
            else:
                with self.assertRaises(InvalidArgumentValueError):
                    result = validate_cidr_fn(tc.dummyclass)
