// Copyright 2019 Yunion
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

package global_vpc

import (
	"context"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GlobalVpcDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(GlobalVpcDeleteTask{})
}

func (self *GlobalVpcDeleteTask) taskFailed(ctx context.Context, gvpc *models.SGlobalVpc, err error) {
	gvpc.SetStatus(ctx, self.UserCred, apis.STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, gvpc, logclient.ACT_DELOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *GlobalVpcDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	gvpc := obj.(*models.SGlobalVpc)
	iVpc, err := gvpc.GetICloudGlobalVpc(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, gvpc)
			return
		}
		self.taskFailed(ctx, gvpc, errors.Wrapf(err, "gvpc.GetICloudGlobalVpc"))
		return
	}
	err = iVpc.Delete()
	if err != nil {
		self.taskFailed(ctx, gvpc, errors.Wrapf(err, "iVpc.Delete"))
		return
	}
	cloudprovider.WaitDeleted(iVpc, time.Second*10, time.Minute*5)
	self.taskComplete(ctx, gvpc)
}

func (self *GlobalVpcDeleteTask) taskComplete(ctx context.Context, gvpc *models.SGlobalVpc) {
	gvpc.RealDelete(ctx, self.GetUserCred())
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    self,
		Action: notifyclient.ActionDelete,
	})
	self.SetStageComplete(ctx, nil)
}
