// Copyright 2016 The kingshard Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

// Copyright 2019 The Gaea Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backend

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/XiaoMi/Gaea/core/errors"
	"github.com/XiaoMi/Gaea/log"
	"github.com/XiaoMi/Gaea/models"
	"github.com/XiaoMi/Gaea/mysql"
	"github.com/XiaoMi/Gaea/util"
)

const (
	weightSplit = "@"

	// DefaultSlice means default slice for namespace
	DefaultSlice = "slice-0"
)

type SlaveStatusCode uint32

const (
	OK   SlaveStatusCode = 1
	DOWN SlaveStatusCode = 2
)

type SlavesInfo struct {
	ConnPool  []ConnectionPool
	Balancer  *balancer
	StatusMap sync.Map
}

// Slice means one slice of the mysql cluster
type Slice struct {
	Cfg models.Slice
	sync.RWMutex

	Master ConnectionPool

	Slave          *SlavesInfo
	StatisticSlave *SlavesInfo

	charset     string
	collationID mysql.CollationID
}

// GetSliceName return name of slice
func (s *Slice) GetSliceName() string {
	return s.Cfg.Name
}

// GetConn get backend connection from different node based on fromSlave and userType
func (s *Slice) GetConn(fromSlave bool, userType int) (pc PooledConnect, err error) {
	if fromSlave {
		if userType == models.StatisticUser {
			pc, err = s.GetSlaveConn(s.StatisticSlave)
			if err != nil {
				return nil, err
			}
		} else {
			pc, err = s.GetSlaveConn(s.Slave)
			if err != nil {
				log.Warn("get connection from slave failed, try to get from master, error: %s", err.Error())
				pc, err = s.GetMasterConn()
			}
		}
	} else {
		pc, err = s.GetMasterConn()
	}
	if err != nil {
		log.Warn("get connection from backend failed, error: %s", err.Error())
		return
	}
	return
}

func (s *Slice) GetDirectConn(addr string) (*DirectConnection, error) {
	return NewDirectConnection(addr, s.Cfg.UserName, s.Cfg.Password, "", s.charset, s.collationID, s.Cfg.Capability)
}

// GetMasterConn return a connection in master pool
func (s *Slice) GetMasterConn() (PooledConnect, error) {
	ctx := context.TODO()
	return s.Master.Get(ctx)
}

func allSlaveIsOffline(SlaveStatusMap *sync.Map) bool {
	var result = true
	SlaveStatusMap.Range(func(k, v interface{}) bool {
		if v == OK {
			result = false
			return false
		}
		return true
	})

	return result
}

// GetSlaveConn get connection from salve
func (s *Slice) GetSlaveConn(slavesInfo *SlavesInfo) (PooledConnect, error) {
	if len(slavesInfo.ConnPool) == 0 || allSlaveIsOffline(&slavesInfo.StatusMap) {
		return nil, errors.ErrNoSlaveDB
	}

	var index int
	// find the idx of the pooledconnect that isn't mark as down
	for size := len(slavesInfo.ConnPool); size > 0; size-- {
		s.Lock()
		index, err := slavesInfo.Balancer.next()
		s.Unlock()
		if err != nil {
			return nil, err
		}

		//We ingore the error, int fact error will never nil
		if value, _ := slavesInfo.StatusMap.Load(index); value == OK {
			break
		}
	}
	ctx := context.TODO()
	return slavesInfo.ConnPool[index].Get(ctx)

}

// Close close the pool in slice
func (s *Slice) Close() error {
	s.Lock()
	defer s.Unlock()
	// close master
	s.Master.Close()

	// close slaves
	for i := range s.Slave.ConnPool {
		s.Slave.ConnPool[i].Close()
	}

	// close statistic slaves
	for i := range s.StatisticSlave.ConnPool {
		s.StatisticSlave.ConnPool[i].Close()
	}

	return nil
}

// ParseMaster create master connection pool
func (s *Slice) ParseMaster(masterStr string) error {
	if len(masterStr) == 0 {
		return errors.ErrNoMasterDB
	}
	idleTimeout, err := util.Int2TimeDuration(s.Cfg.IdleTimeout)
	if err != nil {
		return err
	}
	s.Master = NewConnectionPool(masterStr, s.Cfg.UserName, s.Cfg.Password, "", s.Cfg.Capacity, s.Cfg.MaxCapacity, idleTimeout, s.charset, s.collationID, s.Cfg.Capability)
	return s.Master.Open()
}

// ParseSlave create connection pool of slaves
// (127.0.0.1:3306@2,192.168.0.12:3306@3)
func (s *Slice) ParseSlave(slaves []string) (*SlavesInfo, error) {
	if len(slaves) == 0 {
		return &SlavesInfo{}, nil
	}

	var err error
	var weight int

	count := len(slaves)
	connPool := make([]ConnectionPool, 0, count)
	slaveWeights := make([]int, 0, count)

	//parse addr and weight
	for i := 0; i < count; i++ {
		addrAndWeight := strings.Split(slaves[i], weightSplit)
		if len(addrAndWeight) == 2 {
			weight, err = strconv.Atoi(addrAndWeight[1])
			if err != nil {
				return nil, err
			}
		} else {
			weight = 1
		}
		slaveWeights = append(slaveWeights, weight)
		idleTimeout, err := util.Int2TimeDuration(s.Cfg.IdleTimeout)
		if err != nil {
			return nil, err
		}
		cp := NewConnectionPool(addrAndWeight[0], s.Cfg.UserName, s.Cfg.Password, "", s.Cfg.Capacity, s.Cfg.MaxCapacity, idleTimeout, s.charset, s.collationID, s.Cfg.Capability)
		if err = cp.Open(); err != nil {
			return nil, err
		}
		connPool = append(connPool, cp)
	}

	slaveBalancer := newBalancer(slaveWeights, len(connPool))
	StatusMap := sync.Map{}
	for idx := range connPool {
		StatusMap.Store(idx, OK)
	}

	return &SlavesInfo{connPool, slaveBalancer, StatusMap}, nil
}

// SetCharsetInfo set charset
func (s *Slice) SetCharsetInfo(charset string, collationID mysql.CollationID) {
	s.charset = charset
	s.collationID = collationID
}
