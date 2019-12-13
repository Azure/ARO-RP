# pylint: disable=line-too-long
from azure.cli.core.commands import CliCommandType
from azext_aro._client_factory import cf_aro


def load_command_table(self, _):

    # TODO: Add command type here
    # aro_sdk = CliCommandType(
    #    operations_tmpl='<PATH>.operations#None.{}',
    #    client_factory=cf_aro)


    with self.command_group('aro') as g:
        g.custom_command('create', 'create_aro')
        # g.command('delete', 'delete')
        g.custom_command('list', 'list_aro')
        # g.show_command('show', 'get')
        # g.generic_update_command('update', setter_name='update', custom_func_name='update_aro')


    with self.command_group('aro', is_preview=True):
        pass
