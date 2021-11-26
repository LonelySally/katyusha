// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/katyusha.

package main

import (
	"github.com/gogf/katyusha/.examples/basic/protobuf"
	"github.com/gogf/katyusha/.examples/basic/service"
	"github.com/gogf/katyusha/krpc"
)

func main() {
	c := krpc.Server.NewGrpcServerConfig()
	c.AppId = protobuf.AppId

	s := krpc.Server.NewGrpcServer(c)
	protobuf.RegisterEchoServer(s.Server, new(service.Echo))
	protobuf.RegisterTimeServer(s.Server, new(service.Time))
	s.Run()
}