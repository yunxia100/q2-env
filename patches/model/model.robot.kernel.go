package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"ymlink-q2/plugin"

	"gopkg.in/ini.v1"
)

type RobotKernel struct {
	Objid string

	Hardware     string
	ObjSetupTime string
	SoftWare     string
	Version      string

	Device struct {
		Name string
	}

	Proxy struct {
		Address string `form:"address" bson:"address" json:"address"`
	}

	UserLoginData struct {
		Uis      string
		Uin      int
		Password string
	}

	LoginTime string
}

type RobotKernelPlus struct {
	Objid string

	Hardware     string
	ObjSetupTime string
	SoftWare     string
	Version      string

	Device struct {
		Guid      string
		Imei      string
		QImei16   string
		QImei36   string
		Idfv      string
		Idfa      string
		Name      string
		OsName    string
		OsVersion string

		InternalModel string
		Carrier       string
		MacAddressMd5 string
	}

	Proxy struct {
		Type    string `form:"type" bson:"type" json:"type"`
		Address string `form:"address" bson:"address" json:"address"`
		User    string `form:"user" bson:"user" json:"user"`
		Pwd    string `form:"pwd" bson:"pwd" json:"pwd"`
	}

	UserLoginData struct {
		Uis      string
		Uin      int
		Password string
	}

	LoginTime   string
	LoginTlv119 map[string]string
}

func (robot *Robot) ImportKernel(kernel RobotKernelPlus) (err error) {

	var (
		code    int
		content []byte

		kernel_bytes = []byte{}
	)

	if kernel_bytes, err = json.Marshal(kernel); err != nil {
		return
	}

	kernel_bytes = append(kernel_bytes, 0x0A)

	if code, content, err = robot.Client().PostRow("/batchImport", map[string]string{
		// Headers
	}, map[string]string{
		// params
	}, kernel_bytes); err != nil {
		return
	}

	if code-code%plugin.REQUEST_SUCCESS != plugin.REQUEST_SUCCESS {
		err = fmt.Errorf("code: %d, content: %s", code, string(content))
		return
	}

	if kernel.Objid = string(content); len(kernel.Objid) != 5 {
		err = fmt.Errorf("objid: %s", string(content))
		return
	}

	robot.Kernel.Objid = kernel.Objid

	robot.Kernel.Hardware = kernel.Hardware
	robot.Kernel.ObjSetupTime = kernel.ObjSetupTime
	robot.Kernel.SoftWare = kernel.SoftWare
	robot.Kernel.Version = kernel.Version

	robot.Kernel.Device.Name = kernel.Device.Name

	robot.Kernel.UserLoginData = kernel.UserLoginData

	robot.Kernel.LoginTime = kernel.LoginTime

	return
}

func (robot *Robot) CreateKernel(hardware, software, software_version string) (err error) {

	var (
		code    int
		content []byte
	)

	if code, _, content, err = robot.Client().PostForm("/create", map[string]string{
		// Headers
	}, map[string]string{
		"hardware":        hardware,
		"software":        software,
		"softwareVersion": software_version,
	}, map[string]string{
		// Data
	}); err != nil {
		return
	}

	if code-code%plugin.REQUEST_SUCCESS != plugin.REQUEST_SUCCESS {
		err = fmt.Errorf("code: %d, content: %s", code, string(content))
		return
	}

	if err = json.Unmarshal(content, &robot.Kernel); err != nil {
		err = fmt.Errorf("FAIL：" + string(content))
		return
	}

	return
}

// [PATCH] CreateKernelWithGuid - 创建设备时传入自定义 GUID
func (robot *Robot) CreateKernelWithGuid(hardware, software, software_version, guid_hex string) (err error) {

	var (
		code    int
		content []byte
	)

	params := map[string]string{
		"hardware":        hardware,
		"software":        software,
		"softwareVersion": software_version,
	}

	if guid_hex != "" {
		params["guid_hex"] = guid_hex
	}

	if code, _, content, err = robot.Client().PostForm("/create", map[string]string{
		// Headers
	}, params, map[string]string{
		// Data
	}); err != nil {
		return
	}

	if code-code%plugin.REQUEST_SUCCESS != plugin.REQUEST_SUCCESS {
		err = fmt.Errorf("code: %d, content: %s", code, string(content))
		return
	}

	if err = json.Unmarshal(content, &robot.Kernel); err != nil {
		err = fmt.Errorf("FAIL：" + string(content))
		return
	}

	return
}

func (robot *Robot) PingPong() (result map[string]string, err error) {

	var (
		code    int
		content []byte
	)

	if code, _, content, err = robot.Client().PostForm("/device/pingPong", map[string]string{
		// Headers
	}, map[string]string{
		"objid": robot.Kernel.Objid,
	}, map[string]string{
		// Data
	}); err != nil {
		return
	}

	if code-code%plugin.REQUEST_SUCCESS != plugin.REQUEST_SUCCESS {
		err = fmt.Errorf("code: %d, content: %s", code, string(content))
		return
	}

	if err = json.Unmarshal(content, &result); err != nil {
		err = fmt.Errorf("FAIL：" + string(content))
		return
	}

	return
}

func (robot *Robot) ImportAndroidPack(file_byte []byte) (err error) {

	var (
		code    int
		content []byte
	)

	if code, content, err = robot.Client().PostBinary("/createAndroidFromSandbox", map[string]string{
		// header
	}, map[string]string{
		// param
	}, file_byte); err != nil {
		return
	}

	if code-code%plugin.REQUEST_SUCCESS != plugin.REQUEST_SUCCESS {
		err = fmt.Errorf("code: %d, content: %s", code, string(content))
		return
	}

	if err = json.Unmarshal(content, &robot.Kernel); err != nil {
		err = fmt.Errorf("FAIL：" + string(content))
		return
	}

	return
}

func (robot *Robot) ImportIOSPack(file_byte []byte) (err error) {

	var (
		code    int
		content []byte
	)

	if code, content, err = robot.Client().PostBinary("/createFromICloudBackup", map[string]string{
		// header
	}, map[string]string{
		// param
		"deviceName": "iPhone",
		"carrier":    "Carrier",
	}, file_byte); err != nil {
		return
	}

	if code-code%plugin.REQUEST_SUCCESS != plugin.REQUEST_SUCCESS {
		err = fmt.Errorf("code: %d, content: %s", code, string(content))
		return
	}

	if err = json.Unmarshal(content, &robot.Kernel); err != nil {
		err = fmt.Errorf("FAIL：" + string(content))
		return
	}

	return
}

type RobotIniPack struct {
	QQPassword string `ini:"qqpassword"`
	Token016A  string `ini:"Token016A"`
	Token0106  string `ini:"Token0106"`
	TGTKey     string `ini:"TGTKey"`
	Token010A  string `ini:"Token010A"`
	Token0133  string `ini:"Token0133"`
	Token0134  string `ini:"Token0134"`
	Token0143  string `ini:"Token0143"`
	SessionKey string `ini:"sessionKey"`
}

func (robot *Robot) ImportIni(file_byte []byte) (ini_pack RobotIniPack, err error) {

	var (
		code    int
		content []byte

		uid_str string

		ini_file *ini.File
		section  *ini.Section
		buffer   bytes.Buffer
	)

	file_byte = bytes.ReplaceAll(file_byte, []byte(" "), nil)

	if ini_file, err = ini.Load(file_byte); err != nil {
		return
	}

	for _, section := range ini_file.Sections() {

		if section.Name() != ini.DefaultSection && section.Name() != "" {

			uid_str = section.Name()

			if err = section.MapTo(&ini_pack); err != nil {
				return
			}

			break
		}
	}

	if uid_str == "" {
		err = fmt.Errorf("ini.uid is enpty")
		return
	}

	ini_file = ini.Empty()

	if section, err = ini_file.NewSection(uid_str); err != nil {
		return
	}

	if err = section.ReflectFrom(&ini_pack); err != nil {
		return
	}

	if _, err = ini_file.WriteTo(&buffer); err != nil {
		return
	}

	if code, content, err = robot.Client().PostBinary("/createFrom417ini", map[string]string{
		// header
	}, map[string]string{
		// param
	}, buffer.Bytes()); err != nil {
		return
	}

	if code-code%plugin.REQUEST_SUCCESS != plugin.REQUEST_SUCCESS {
		err = fmt.Errorf("code: %d, content: %s", code, string(content))
		return
	}

	if err = json.Unmarshal(content, &robot.Kernel); err != nil {
		err = fmt.Errorf("FAIL：" + string(content))
		return
	}

	return
}
