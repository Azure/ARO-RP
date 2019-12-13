from knack.util import CLIError


def create_aro(cmd, resource_group_name, aro_name, location=None, tags=None):
    raise CLIError('TODO: Implement `aro create`')


def list_aro(cmd, resource_group_name=None):
    raise CLIError('TODO: Implement `aro list`')


def update_aro(cmd, instance, tags=None):
    with cmd.update_context(instance) as c:
        c.set_param('tags', tags)
    return instance
