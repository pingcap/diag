import argparse
from ansible.parsing.dataloader import DataLoader
from ansible.inventory.manager import InventoryManager


def ping():
    loader = DataLoader()
    inventory = InventoryManager(loader=loader, sources='localhost,')

def collect_basic():
    print "basic"

def collect_software():
    print "software"

def collect_hardware():
    print "hardware"

def collect_dbinfo():
    print "dbinfo"

def collect_resource():
    print "resource"

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--inspection-id", type=str, required=True, help="inspection identity")
    parser.add_argument("--data-dir", type=str, required=True, help="data directory")
    parser.add_argument("--inventory", type=str, required=True, help="inventory file")
    parser.add_argument("--topology", type=str, required=True, help="topology.json")
    parser.add_argument("--collect", type=str, required=True, help="items to collect")

    args = parser.parse_args()

    items = map(lambda x: x.split(':'), args.collect.split(','))
    for item in items:
        if item[0] == "basic":
            collect_basic()
        elif item[0] == "software":
            collect_software()
        elif item[0] == "hardware":
            collect_hardware()

if __name__ == "__main__":
    main()