current = "0.4.0"
legacy = ["0.1.0", "0.2.0"]
all_versions = ["0.1.0", "0.2.0", "0.3.0", "0.3.1", "0.4.0"]


def parse(version):
    l = [int(x, 10) for x in version.split('.')]
    l.reverse()
    return sum(x * (10 ** i) for i, x in enumerate(l))


def greater_than_or_equal_to(version, other_version):
    version = parse(version)
    other_version = parse(other_version)

    return version > other_version


def build_string(cni_name, version):
    return "CNI %s plugin version %s" % (cni_name, version)
