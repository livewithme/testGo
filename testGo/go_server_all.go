package main

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/seehuhn/mt19937"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	/*显示主信息*/
	http.HandleFunc("/index", index)
	//根目录获取显示文件 http.Handle("/", http.FileServer(http.Dir("./")))
	//需要限制不是根目录访问的 使用下面的方法
	http.Handle("/tmpfiles/", http.StripPrefix("/tmpfiles/", http.FileServer(http.Dir("./tmp"))))
	http.HandleFunc("/uploadfile", fileUploadHandle)
	http.HandleFunc("/getImg", downloadHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.ListenAndServe(":8080", nil)

}

/**
 * @Author:      YaoWang
 * @DateTime:    2017-04-21 17:09:07
 * @Description: 下载图片数据
 */
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	//解析请求 获取imgid
	r.ParseForm()
	imageid := r.Form["imgid"][0]
	if len(imageid) != 16 {
		w.Write([]byte("Error:ImageID incorrect."))
		return
	}
	imgpath := ImageID2Path(imageid)
	if !FileExist(imgpath) {
		w.Write([]byte("Error:Image Not Found."))
		return
	}
	http.ServeFile(w, r, imgpath)
}

/**
 * @Author:      YaoWang
 * @DateTime:    2017-04-21 17:08:46
 * @Description: 上传图片图片数据
 */
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	//加了头信息 ，既可以访问 跨域请求（解决方法）
	w.Header().Add("Access-Control-Allow-Origin", "*")
	//随机生成一个不存在的fileid
	var imgid string
	for {
		imgid = MakeImageID()
		if !FileExist(ImageID2Path(imgid)) {
			break
		}
	}
	//上传参数为uploadfile
	r.ParseMultipartForm(32 << 20)
	file, _, err := r.FormFile("uploadfile")
	if err != nil {
		log.Println(err)
		w.Write([]byte("Error:Upload Error."))
		return
	}
	defer file.Close()
	//检测文件类型
	buff := make([]byte, 512)
	_, err = file.Read(buff)
	if err != nil {
		log.Println(err)
		w.Write([]byte("Error:Upload Error."))
		return
	}
	filetype := http.DetectContentType(buff)
	if filetype != "image/jpeg" {
		w.Write([]byte("Error:Not JPEG."))
		return
	}
	//回绕文件指针
	log.Println(filetype)
	if _, err = file.Seek(0, 0); err != nil {
		log.Println(err)
	}
	//提前创建整棵存储树（如果不进行存储树结构创建，下面的文件创建不会成功）
	if err = BuildTree(imgid); err != nil {
		log.Println(err)
	}
	//将文件写入ImageID指定的位置
	f, err := os.OpenFile(ImageID2Path(imgid), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Println(err)
		w.Write([]byte("Error:Save Error."))
		return
	}
	defer f.Close()
	io.Copy(f, file)
	// w.Write([]byte(imgid))
	data := photo_info{
		result{100, "上传成功!"},
		imgid,
	}
	//返回JSON格式数据 json.Marshal返回的是bytes
	returnData, err := json.Marshal(data)
	//解码JSON方法是json.Unmarshal返回一个对象结构体
	fmt.Fprintf(w, string(returnData))
}

/**
 * @Author:      YaoWang
 * @DateTime:    2017-04-21 16:47:50
 * @Description: 文件上传处理事件
 */
func fileUploadHandle(w http.ResponseWriter, r *http.Request) {
	//首先解析请求
	r.ParseMultipartForm(32 << 20)
	//获取键值uoloadfile对应的文件
	file, handler, err := r.FormFile("uploadfile")
	if err != nil {
		fmt.Println(err)
		return
	}
	//延时关闭文件
	defer file.Close()
	//创建或者读取文件 此处文件名是个争议
	f, err := os.OpenFile(handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
	//延时关闭文件
	defer f.Close()
	//复制文件到相应地址
	io.Copy(f, file)
	data := result{
		100,
		"上传成功!",
	}
	returnData, err := json.Marshal(data)
	fmt.Fprintln(w, string(returnData))
}

/**
 * @Author:      YaoWang
 * @DateTime:    2017-04-21 16:01:27
 * @Description: 显示主页面 返回JOSN数据
 */
func index(w http.ResponseWriter, r *http.Request) {
	//加了头信息 ，既可以访问 跨域请求（解决方法）
	w.Header().Add("Access-Control-Allow-Origin", "*")
	//r.Method 判断是什么请求类型
	if r.Method == "GET" {
		fmt.Println("hello, world 这是get请求！！")
	} else {
		fmt.Println("hello, world 这是其他形式请求！！")
	}
	r.ParseForm()
	// fmt.Println(r.Form["username"])
	//解析传过来的参数
	// fmt.Println(r.URL.RawQuery)
	queryForm, err := url.ParseQuery(r.URL.RawQuery)
	if err == nil && len(queryForm["name"]) > 0 {
		fmt.Println(r.Form["name"])
	}
	data := user_info{
		result{100, "获取成功!"},
		"这是我的全服务器!",
		"一定可以实现的!",
	}
	//返回JSON格式数据 json.Marshal返回的是bytes
	returnData, err := json.Marshal(data)
	//解码JSON方法是json.Unmarshal返回一个对象结构体
	fmt.Fprintf(w, string(returnData))
}

/*返回的参数JSON化的必须大写开头*/
type result struct {
	ReturnCode int
	Msg        string
}

/*用户信息*/
type user_info struct {
	result
	Name string
	Code string
}

/*头像信息*/
type photo_info struct {
	result
	Imgid string
}

/**
 * @Author:      YaoWang
 * @DateTime:    2017-04-21 17:09:28
 * @Description: 随机生成文件ID
 */
func MakeImageID() string {
	mt := mt19937.New()
	mt.Seed(time.Now().UnixNano())
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, mt.Uint64())
	return strings.ToUpper(hex.EncodeToString(buf))
}

/**
 * @Author:      YaoWang
 * @DateTime:    2017-04-21 17:09:51
 * @Description: id转为地址
 */
func ImageID2Path(imageid string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s/%s.jpg", "./img/", imageid[0:2], imageid[2:4], imageid[4:6], imageid[6:8], imageid[8:10], imageid[10:12], imageid[12:14], imageid[14:16])
}

/**
 * @Author:      YaoWang
 * @DateTime:    2017-04-21 17:20:52
 * @Description: 判断文件是否存在
 */
func FileExist(filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		return false
	} else {
		return true
	}
}

/**
 * @Author:      YaoWang
 * @DateTime:    2017-04-21 17:21:11
 * @Description: 创建文件树
 */
func BuildTree(imageid string) error {
	return os.MkdirAll(fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s", "./img/", imageid[0:2], imageid[2:4], imageid[4:6], imageid[6:8], imageid[8:10], imageid[10:12], imageid[12:14]), 0666)
}
