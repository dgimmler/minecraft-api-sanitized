from time import time
from math import floor
import json
import requests
import os

# Global config
# location of log files
LOG_DIR = "/home/ec2-user/MinecraftServer/SpigotMC/logs"
# LOG_DIR = "."
LOG_PATH = f"{LOG_DIR}/latest.log"  # Spigot log file
PYTHON_LOG_PATH = f"{LOG_DIR}/log_handling.log"  # log file for this script
API_VERSION = "v1"  # current version of API (sort key for main table)
# base url for API
API_ENDPOINT = f"https://16z7hps25k.execute-api.us-west-2.amazonaws.com/{API_VERSION}"
# headers to use for all request, mainly needs api key
HEADERS = {"x-api-key": "5nVd9RSnRt6Idry8upQpr1YBwc6N52mZ3NkpycDj"}


def get_last_login(username):
    """Returns dynamodb item for most recent login as python dict
    """
    print(f"Getting last login session for {username}...")
    endpoint = f"{API_ENDPOINT}/getLogins"
    data = json.dumps({'Usernames': [username]})
    response = requests.post(endpoint, headers=HEADERS, data=data)
    if str(response.content.decode("UTF-8")) == 'null' or str(response.status_code)[0] != '2':
        return {}  # if this is a first time login, return nothing
    return json.loads(response.content.decode("UTF-8"))[0]


def get_users():
    """Returns dynamodb item for most recent login as python dict
    """
    print(f"Getting last login sessions for users...")
    endpoint = f"{API_ENDPOINT}/getLogins"
    response = requests.post(endpoint, headers=HEADERS)
    if str(response.content.decode("UTF-8")) == 'null' or str(response.status_code)[0] != '2':
        return []  # if this is a first time login, return empty list
    return json.loads(response.content.decode("UTF-8"))


def logout_user(username, login_time, version=API_VERSION):
    """Sets logout time for open login session
    """
    if version == API_VERSION:
        print(f"Marking user {username} as logged out...")
    endpoint = f"{API_ENDPOINT}/upsertLogin"
    data = json.dumps({'Username': username,
                       'Version': version,
                       'LoginTime': login_time,
                       'LogoutTime': floor(time())})
    response = requests.post(endpoint, headers=HEADERS, data=data)
    if str(response.status_code)[0] == '2':
        if version == API_VERSION:
            return logout_user(username, login_time, str(floor(time())))
        return get_last_login(username)
    return json.loads(response.content.decode("UTF-8").replace('null', 'None'))


def logout_users():
    print("Server stopping... marking any logged in users as logged out")
    users = get_users()
    count = 0
    for user in users:
        if not user.get('LogoutTime'):
            count += 1
            logout_user(user.get('Username'), user.get('LoginTime'))
    if count == 0:
        print("All users already logged out. No action taken.")
    return


def lambda_handler(event, context):
    print('hello')
    logout_users()
    print('hi')
    headers = {"Access-Control-Allow-Origin": os.environ.get("CloudfrontOrigin"),
               "Access-Control-Allow-Headers:": "*"}
    print('hey')
    r = json.dumps({"statusCode": 200, "body": "success", "headers": headers})
    print(r)
    return r
