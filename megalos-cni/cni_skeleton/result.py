import json
import sys

# A really raw result handling, you should create dicts with values outside this file.
# TODO: This class should be done good (with 0.2.0 support)
# TODO: See https://github.com/containernetworking/cni/blob/master/pkg/types/current/types.go
def create_result(cni_version, interfaces=None, ips=None, routes=None, dns=None):
    result = {
        "cniVersion": cni_version
    }

    if interfaces is not None:
        result["interfaces"] = interfaces
    if ips is not None:
        result["ips"] = ips
    if routes is not None:
        result["routes"] = routes
    if dns is not None:
        result["dns"] = dns

    return result


def create_error(code, msg, details=None):
    error = {
        "code": code,
        "msg": msg
    }

    if details is not None:
        error["details"] = details

    return error


def print_result(result):
    sys.stdout.write(json.dumps(result, indent=True))

    return None
