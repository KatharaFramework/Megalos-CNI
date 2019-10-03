import sys
import os
import json
import version
import result


# Base code is took from:
# https://github.com/containernetworking/cni/blob/master/pkg/skel/skel.go#L300
def get_cmd_args_from_env():
    env_vars = {
        "CNI_COMMAND": {
            "value": None,
            "requiredForCmd": ["ADD", "CHECK", "DEL"]
        },
        "CNI_CONTAINERID": {
            "value": None,
            "requiredForCmd": ["ADD", "CHECK", "DEL"]
        },
        "CNI_NETNS": {
            "value": None,
            "requiredForCmd": ["ADD", "CHECK"]
        },
        "CNI_IFNAME": {
            "value": None,
            "requiredForCmd": ["ADD", "CHECK", "DEL"]
        },
        "CNI_ARGS": {
            "value": None,
            "requiredForCmd": []
        },
        "CNI_PATH": {
            "value": None,
            "requiredForCmd": ["ADD", "CHECK", "DEL"]
        }
    }

    args_missing = []

    for var in env_vars:
        value = os.environ.get(var)

        if not value:
            if env_vars["CNI_COMMAND"]["value"] in env_vars[var]["requiredForCmd"] or var == "CNI_COMMAND":
                args_missing.append(var)
        else:
            env_vars[var]["value"] = value

    if len(args_missing) > 0:
        return "", None, "Required env variables [%s] missing" % (",".join(args_missing))

    config = ""
    if env_vars["CNI_COMMAND"]["value"] != "VERSION":
        for line in sys.stdin:
            config += line

    cmd_args = {}
    for var in env_vars:
        if var != "CNI_COMMAND":
            cmd_args[var] = env_vars[var]["value"]
    cmd_args["stdin"] = config

    return env_vars["CNI_COMMAND"]["value"], cmd_args, None


def create_typed_error(msg):
    return result.create_error(100, msg)


def check_version_and_call(args, version_info, func):
    config_version = args["stdin"]["cniVersion"] if args["stdin"]["cniVersion"] else "0.1.0"

    if config_version not in version_info:
        return result.create_error(
            code=1,
            msg="Incompatible CNI versions.",
            details="Config is %s, plugin supports %s" % (config_version, str(version_info))
        )

    return func(args)


def validate_config(config):
    parsed_config = json.loads(config)
    if not parsed_config["name"]:
        return None, "Missing network name."

    return parsed_config, None


def cmd_check_proxy(args, version_info, cmd_check):
    config_version = args["stdin"]["cniVersion"] if args["stdin"]["cniVersion"] else "0.1.0"
    if not version.greater_than_or_equal_to(config_version, "0.4.0"):
        return result.create_error(
            code=2,
            msg="Config version does not allow CHECK."
        )

    for plugin_version in version_info:
        if version.greater_than_or_equal_to(plugin_version, config_version):
            return check_version_and_call(args, version_info, cmd_check)

    return result.create_error(
        code=2,
        msg="Plugin version does not allow CHECK."
    )


def cmd_version(version_info):
    sys.stdout.write(json.dumps({
        "cniVersion": version.current,              # Current version of this script!
        "supportedVersions": version_info           # Supported versions by the plugin
    }))

    return None


def plugin_main_with_error(cmd_add, cmd_check, cmd_del, version_info, about):
    cmd, args, err = get_cmd_args_from_env()

    # Print the about string to stderr when no command is set
    if err:
        if not os.environ.get("CNI_COMMAND") and about is not None:
            sys.stderr.write(about + "\n")
            return None

        return create_typed_error("Error while reading args.")

    if cmd != "VERSION":
        parsed_config, err = validate_config(args["stdin"])
        if err:
            return create_typed_error(err)
        args["stdin"] = parsed_config

    if cmd == "ADD":
        return check_version_and_call(args, version_info, cmd_add)
    elif cmd == "CHECK":
        return cmd_check_proxy(args, version_info, cmd_check)
    elif cmd == "DEL":
        return check_version_and_call(args, version_info, cmd_del)
    elif cmd == "VERSION":
        return cmd_version(version_info)
    else:
        return create_typed_error("Unknown CNI_COMMAND: %s" % cmd)


def plugin_main(cmd_add, cmd_check, cmd_del, version_info, about=None):
    try:
        err = plugin_main_with_error(cmd_add, cmd_check, cmd_del, version_info, about)
    except Exception as e:
        err = create_typed_error(e.message)

    if err:
        result.print_result(err)
        sys.exit(1)
