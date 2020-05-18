# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

import unittest
import mock
from azext_aro.custom import generate_random_id


@mock.patch('azext_aro.custom.random.choice', return_value='r')
class TestGenerateRandomIdHelper(unittest.TestCase):
    def test_random_id_length(self, mock_random_id):
        random_id = generate_random_id()
        self.assertTrue(mock_random_id.called_once)
        self.assertEqual(len(random_id), 8)

    def test_random_id_starts_with_letter(self, mock_random_id):
        random_id = generate_random_id()
        self.assertTrue(mock_random_id.called_once)
        self.assertTrue(random_id[0].isalpha())

    def test_random_id_is_alpha_num(self, mock_random_id):
        random_id = generate_random_id()
        self.assertTrue(mock_random_id.called_once)
        self.assertTrue(random_id.isalnum())
