import logging

from configs.logger import setup_logger
from configs.db import Database


# setup logger
setup_logger()

def main():
    logging.info("Starting Xarvis...")
    db = Database()


if __name__ == "__main__":
    main()
