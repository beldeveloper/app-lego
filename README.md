# App Lego

## Description

The idea of this project is to create a tool that lets building the 3rd party applications using specific VCS branches.
Such an approach provides the QA team with the possibility to test a particular feature separately from the others.

## Technical requirements

The application requires Postgres and Git.

## Environment variables

```
APP_LEGO_HTTP_PORT
APP_LEGO_HTTPS_CRT
APP_LEGO_HTTPS_KEY
APP_LEGO_REPOS_DIR
APP_LEGO_DB_HOST
APP_LEGO_DB_PORT
APP_LEGO_DB_USER
APP_LEGO_DB_PASSWORD
APP_LEGO_DB_NAME
APP_LEGO_HOOK_HANDLER_ADDR
APP_LEGO_ACCESS_KEY
```