package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/o1egl/govatar"
	"gorm.io/gorm"
	ew "ku-chat/internal/entity/web"
	es "ku-chat/internal/entity/ws"
	"ku-chat/internal/model"
	"ku-chat/internal/service"
	"ku-chat/internal/websocket"
	"ku-chat/pkg/config"
	"ku-chat/pkg/utils/encrypt"
	"log"
	"net/http"
	"os"
	"time"
)

// Login 用户注册
func Register(ctx *gin.Context) {
	s := service.Context(ctx)
	if ctx.Request.Method == http.MethodGet {
		s.View("register", nil)
		return
	}

	var req ew.RegisterReq
	if err := ctx.ShouldBind(&req); err != nil {
		s.Back().WithError(err).Redirect()
		return
	}
	if err := g.Validator().Data(req).Run(context.Background()); err != nil {
		s.Back().WithError(err.FirstError()).Redirect()
		return
	}

	var user *model.Users
	err := model.User().M.Where("name = ?", req.Name).Find(&user).Error
	if err != nil {
		s.Back().WithError(err).Redirect()
		return
	}
	if user.ID > 0 {
		s.Back().WithError("用户名已被注册，请更换用户名继续尝试").Redirect()
		return
	}

	avatar, err := genAvatar(req.Name)
	if err != nil {
		s.Back().WithError("用户默认头像生成失败").Redirect()
		return
	}

	res := model.User().M.Create(&model.Users{
		Name:     req.Name,
		Avatar:   avatar,
		Password: encrypt.GenerateFromPassword(req.Password),
	})
	if res.Error != nil || res.RowsAffected <= 0 {
		s.Back().WithError("用户注册失败，请稍后在试").Redirect()
	} else {
		s.To("/login").WithMsg("注册成功，请继续登录").Redirect()
	}
}

func genAvatar(name string) (string, error) {
	path := fmt.Sprintf("%s/users/", config.Conf.Upload.Path)

	// 检查目录是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_ = os.Mkdir(path, os.ModePerm)
		_ = os.Chmod(path, os.ModePerm)
	}

	avatarName := encrypt.Md5(gconv.String(time.Now().UnixMicro()))
	avatarPath := fmt.Sprintf("users/%s.png", avatarName)
	uploadPath := fmt.Sprintf("%s/%s", config.Conf.Upload.Path, avatarPath)

	if err := govatar.GenerateFileForUsername(1, name, uploadPath); err != nil {
		log.Println(err)
		return "", err
	} else {
		return "/assets/upload/" + avatarPath, nil
	}
}

// Login 用户登录
func Login(ctx *gin.Context) {
	s := service.Context(ctx)
	if ctx.Request.Method == http.MethodGet {
		s.View("login", nil)
		return
	}

	var req ew.LoginReq
	if err := ctx.ShouldBind(&req); err != nil {
		s.Back().WithError(err).Redirect()
		return
	}

	var user model.Users
	err := model.User().M.Where("name = ?", req.Name).Find(&user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.Back().WithError(err).Redirect()
		return
	}
	if user.ID <= 0 || !encrypt.CompareHashAndPassword(user.Password, req.Password) {
		s.Back().WithError("用户名或密码错误").Redirect()
	} else {
		s.SetAuth(user)
		s.To("/").WithMsg("登录成功").Redirect()
	}
}

// Logout 退出登录
func Logout(ctx *gin.Context) {
	s := service.Context(ctx)

	s.Forget()

	s.To("/login").WithMsg("退出成功").Redirect()
}

// Search 用户搜索
func Search(ctx *gin.Context) {
	s := service.Context(ctx)
	keys := ctx.Query("user")
	if len(keys) <= 0 {
		s.Json(gin.H{"code": 1, "msg": "请输入关键词"})
		return
	}

	var user *model.Users
	res := model.User().M.Where("name like ?", fmt.Sprintf("%%%s%%", keys)).Or("id = ?", keys).Find(&user)
	if res.Error != nil {
		s.Json(gin.H{"code": 1, "msg": res.Error})
		return
	}
	if user.ID <= 0 {
		s.Json(gin.H{"code": 1, "msg": "用户不存在"})
	} else {
		s.Json(gin.H{"code": 0, "msg": "ok", "data": user})
	}
}

func AddFriend(ctx *gin.Context) {
	s := service.Context(ctx)

	var req ew.AddReq
	if err := ctx.ShouldBind(&req); err != nil {
		s.Json(gin.H{"code": 1, "msg": err.Error()})
		return
	}
	if err := g.Validator().Data(req).Run(context.Background()); err != nil {
		s.Json(gin.H{"code": 1, "msg": err.Error()})
		return
	}

	var record *model.Records
	res := model.Record().M.Where("user_id", s.Auth().ID).Where("target_id", req.TargetID).Find(&record)
	if res.Error != nil {
		s.Json(gin.H{"code": 1, "msg": res.Error})
		return
	}

	log.Println(req)
	if record.ID > 0 {
		res = model.Record().M.Where("id", record.ID).Updates(&model.Records{
			Remark:   req.Remark,
			State:    0,
			ReadedAt: nil,
		})
	} else {
		res = model.Record().M.Create(&model.Records{
			UserId:   gconv.Int64(s.Auth().ID),
			TargetId: gconv.Int64(req.TargetID),
			Remark:   req.Remark,
			State:    0,
		})
	}

	if res.Error != nil || res.RowsAffected <= 0 {
		s.Json(gin.H{"code": 1, "msg": "申请添加好友失败"})
		return
	}

	conn, err := websocket.Core.Get(gconv.String(req.TargetID))
	if err != nil {
		s.Json(gin.H{"code": 1, "msg": "申请添加好友失败"})
		return
	}

	wsData := es.AddFriendWsReq{}

	wsData.ID = 2
	wsData.Data.User = s.Auth()
	wsData.Data.Remark = req.Remark

	byteData, _ := json.Marshal(wsData)

	conn.Conn.WriteMessage(1, byteData)

	s.Json(gin.H{"code": 0, "msg": "申请添加好友成功"})
}