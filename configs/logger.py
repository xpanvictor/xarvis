import logging

def setup_logger():
    logging.basicConfig(filename='logs/xarvis.log', level=logging.INFO,
                        format='%(asctime)s:%(levelname)s: %(message)s')
