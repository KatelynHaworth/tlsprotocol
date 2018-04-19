TLS Protocol Listener [![Build Status](https://travis-ci.org/LiamHaworth/tlsprotocol.svg)](https://travis-ci.org/LiamHaworth/tlsprotocol) [![Tags](https://img.shields.io/github/tag/LiamHaworth/tlsprotocol.svg)](https://github.com/LiamHaworth/tlsprotocol/tags) [![GoDoc](https://godoc.org/github.com/LiamHaworth/tlsprotocol?status.svg)](https://godoc.org/github.com/LiamHaworth/tlsprotocol)
=====================

TLS Protocol Listener provides an abstraction on top of the TLS listener functionality to provide the ability to have
individual net.Listeners for application layer protocols (ALPN) negotiated during the TLS handshake between the client
and server.

Installing
==========

```sh
dep ensure -add github.com/LiamHaworth/tlsprotocol
```

or

```sh
go get github.com/LiamHaworth/tlsprotocol
```

Contributing
=============

To contribute to this project, please follow this guide:

  1. Create an issue detailing your planned contribution
  2. Fork this repository and implement your contribution
  3. Create a pull request linking back to the issue
  4. Await approval and merging

Copyright, licence and authors
==============================

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.


| Author      | Email                | Copyright                                        |
|:------------|:---------------------|:-------------------------------------------------|
| Liam Haworth| liamh@familyzone.com | Copyright (c) Family Zone Cyber Safety Ltd, 2018 |
