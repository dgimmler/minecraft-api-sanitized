# Purpose

This is the public, sanitized version of the (private) repository. Some specific instance IDs and ARNs have been removed.

This was a project for me to familiarize myself and test some features of SAM templates, Golang and API Gateway. Most technical decisions were made more in respect to the tools I wanted exposure to than for the best tool for the job.

# Overview

This SAM template deploys an API over API Gateway that includes several endpoints used for managing a minecraft server and website. The endpoints are largely called from the Minecraft website.

All endpoints proxy to lambda functions written in Go. The API structure is very basic. The endpoints are organized in a flat structure under a single version tree:

```
v1/
    /getKey
    /getLogins
    /getServerStatus
    /getServerTime
    /logoutUsers
    /markServerStarted
    /startServer
    /stopServer
    /updateTimer
    /upsertLogin
```

## /getKey

Returns the correct API key needed for other API calls. Mainly used as a process to store the key "on the server" for the serverless website.

## /getLogins

Returns either list of the latest login times for all users who have ever logged into the minecraft server or a list of all logins for a single user, depending on parameters passed.

## /getServerStatus

The EC2 instance running the minecraft server and the minecraft server service have separate statusesf, as the minecraft server service isn't started until the EC2 instance is fully booted up. This call returns the status of the actual minecraft server service (started, stopped). If the EC2 instance is starting or stopping, it returns starting or stopping accordingly.

## /getServerTime

The server start event starts a timer for 2 hours after which the server will automatically shut off (to save costs). This call returns how much time is left on that timer.

## /logoutUsers

A dynmamodb table tracks the login and logout times for all users who have logged into the minecraft server. This call will mark any currently logged in users as logged out and set their logout times to the current time. Mainly called when the servdr shuts off.

## /markServerStarted

The server status returned by /getServerStatus is stored in a parameter store value. This call marks that parameter as started. Mainly called directly from the EC2 instance once the minecraft server service is seen as running.

## /startServer

As the name suggests, starts the minecraft server, starting the EC2 instance and in turn starting the minecraft server service.

## /stopServer

As the name suggests,s tops the minecraft server, gracefully stopping the minecraft server service, turning off the EC2 instance and taking a snapshot once stopped.

## /updateTimer

The shutdown time for the server is stored in a parameter store value that. While /getServerTimer returns the number of seconds between now and that shut down time, this call sets that shutdown time. Either that is 2 hours from now if the server is starting or 30 minutes from the current shutdown time (up to two hours from now) depending on parameters passed.

## /upsertLogin

Updates or creates a new login session for a user logged into the minecraft server. This essentially means an entry in the dynamodb table. Items in the table simply track the login and logout times. This call either creates that item, or updates the login/logout time as needed.
