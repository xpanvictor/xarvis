# db conn utilities
import sqlite3
from sqlite3 import Connection, Cursor
import constants


class Database:
    connection: Connection | None = None
    def __init__(self):
        db_url = constants.DB_URL
        if db_url is None:
            raise ValueError("DB_URL cannot be None")
        connection = sqlite3.connect(db_url)


    @staticmethod
    def get_conn_cursor() -> Cursor:
        return Database.connection.cursor()

