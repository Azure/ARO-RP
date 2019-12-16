from azext_aro._client_factory import cf_aro
from azext_aro._params import load_arguments
from azext_aro.commands import load_command_table
from azure.cli.core import AzCommandsLoader
from azure.cli.core.commands import CliCommandType


class AroCommandsLoader(AzCommandsLoader):
    def __init__(self, cli_ctx=None):
        aro_custom = CliCommandType(operations_tmpl='azext_aro.custom#{}',
                                    client_factory=cf_aro)
        super(AroCommandsLoader, self).__init__(cli_ctx=cli_ctx,
                                                custom_command_type=aro_custom)

    def load_command_table(self, args):
        load_command_table(self, args)
        return self.command_table

    def load_arguments(self, command):
        load_arguments(self, command)


COMMAND_LOADER_CLS = AroCommandsLoader
