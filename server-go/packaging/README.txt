VXA-OS server test package
==========================

Requirements
------------
- Windows 64-bit
- The client must point to this machine's IP and port in its Configs.
- Allow the port in Windows Firewall if the client is on another machine.

First run
---------
1. Open a terminal in this folder (or double-click start-server.bat).
2. Create an account and a character:

   vxaos-admin.exe account create -config config.toml -user admin -pass secret -group 2
   vxaos-admin.exe character create -config config.toml -user admin -name Hero -class 1 -sex 0

3. Start the server:

   vxaos-server.exe -config config.toml

The database is created automatically in data\vxaos.db on first start.

Folders
-------
- config.toml   server bootstrap and database settings
- game-data\    RPG Maker data used by the server
- content\      shared configs, enums, locales and quests
- data\         SQLite database (created automatically; do not ship it)
- vxaos-server.exe
- vxaos-admin.exe

Connecting
----------
The server listens on 0.0.0.0:5000 by default. The client needs the same
GAME_VERSION as the server (currently 12983) and the protocol v2 client patch.
If the client is on another PC, set its HOST to the server's LAN IP and PORT to
5000.

Admin commands
--------------
- account create: creates a login account
- character create: creates a character in slot 0
- character list: lists an account's characters
- character delete: deletes a character

Use -group 2 for an administrator account.
