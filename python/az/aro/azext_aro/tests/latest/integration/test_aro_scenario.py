# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

import os
from unittest import mock

from azure_devtools.scenario_tests import AllowLargeResponse
from azure.cli.testsdk import ResourceGroupPreparer
from azure.cli.testsdk import ScenarioTest


TEST_DIR = os.path.abspath(os.path.join(os.path.abspath(__file__), '..'))


class AroScenarioTest(ScenarioTest):
    @ResourceGroupPreparer(name_prefix='cli_test_aro')
    def test_aro(self, resource_group):
        self.kwargs.update({
            'name': 'test1'
        })

        # test aro create
        with mock.patch('azure.cli.command_modules.aro._rbac._gen_uuid', side_effect=self.create_guid):
            self.cmd('aro create -g {rg} -n {name} --tags foo=doo', checks=[
                self.check('tags.foo', 'doo'),
                self.check('name', '{name}')
            ])

        # Test aro_validate_permissions
        with mock.patch('azure.cli.command_modules.aro._rbac._gen_uuid', side_effect=self.create_guid):
            validated_permissions_output = self.cmd('aro validate_permissions -g {rg}')
            self.assertTrue(validated_permissions_output, '')

        self.cmd('aro update -g {rg} -n {name} --tags foo=boo', checks=[
            self.check('tags.foo', 'boo')
        ])

        count = len(self.cmd('aro list').get_output_in_json())
        self.cmd('aro show -g {rg} -n {name}', checks=[
            self.check('name', '{name}'),
            self.check('resourceGroup', '{rg}'),
            self.check('tags.foo', 'boo')
        ])

        self.cmd('aro delete -g {rg} -n {name}')

        final_count = len(self.cmd('aro list').get_output_in_json())
        self.assertTrue(final_count, count - 1)
