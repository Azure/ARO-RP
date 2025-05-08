# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

from azext_aro._client_factory import cf_aro
from azext_aro._params import load_arguments
from azext_aro.commands import load_command_table
from azure.cli.core import (
    AzCommandsLoader,
    ModExtensionSuppress
)
from azure.cli.core.commands import CliCommandType
from azure.cli.core.aaz import load_aaz_command_table
try:
    from . import aaz
except ImportError:
    aaz = None


class AroCommandsLoader(AzCommandsLoader):
    def __init__(self, cli_ctx=None):
        aro_custom = CliCommandType(
            operations_tmpl='azext_aro.custom#{}',
            client_factory=cf_aro)
        suppress = ModExtensionSuppress(__name__, 'aro', '1.0.0',
                                        reason='Its functionality is included in the core az CLI.',
                                        recommend_remove=True)
        super().__init__(cli_ctx=cli_ctx,
                         suppress_extension=suppress,
                         custom_command_type=aro_custom)

    def load_command_table(self, args):
        if aaz:
            load_aaz_command_table(
                loader=self,
                aaz_pkg_name=aaz.__name__,
                args=args
            )
        load_command_table(self, args)
        return self.command_table

    def load_arguments(self, command):
        load_arguments(self, command)


COMMAND_LOADER_CLS = AroCommandsLoader
