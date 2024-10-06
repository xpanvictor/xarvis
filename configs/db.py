# db conn utilities
import sqlite3
from sqlite3 import Connection, Cursor
import constants


class Database:
    """
    Singleton class for db conn
    """
    instance = None
    connection: Connection | None = None

    def __new__(cls):
        if not hasattr(cls, 'instance'):
            cls.instance = super(Database, cls).__new__(cls)
        return cls.instance

    def __init__(self):
        db_url = constants.DB_URL
        if db_url is None:
            raise ValueError("DB_URL cannot be None")
        self.connection = sqlite3.connect(db_url)


    def get_conn_cursor(self) -> Cursor:
        return self.connection.cursor()

    def close(self):
        self.connection.close()

