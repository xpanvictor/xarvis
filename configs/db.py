# db conn utilities
import sqlite3
from sqlite3 import Connection, Cursor
from configs import constants


class Database:
    """
    Singleton class for DB conn
    Interesting concept I picked from JS world
    """
    instance = None
    _connection: Connection | None = None

    def __new__(cls):
        if cls.instance is None:
            cls.instance = super().__new__(cls)
        return cls.instance

    def __init__(self):
        db_url = constants.DB_URL
        if db_url is None:
            raise ValueError("DB_URL cannot be None")
        self.connection = sqlite3.connect(db_url)


    @property
    def connection(self) -> Connection:
        if not hasattr(self, '_connection'):
            raise RuntimeError("Database instance has not been initialized")
        return self._connection

    @connection.setter
    def connection(self, connection: Connection):
        self._connection = connection

    @property
    def cursor(self) -> Cursor:
        return self.connection.cursor()

    def close(self):
        self.connection.close()

    def __del__(self):
        self.connection.close()

