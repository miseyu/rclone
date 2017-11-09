package all

import (
	// Active file systems
	_ "github.com/miseyu/rclone/amazonclouddrive"
	_ "github.com/miseyu/rclone/azureblob"
	_ "github.com/miseyu/rclone/b2"
	_ "github.com/miseyu/rclone/box"
	_ "github.com/miseyu/rclone/crypt"
	_ "github.com/miseyu/rclone/drive"
	_ "github.com/miseyu/rclone/dropbox"
	_ "github.com/miseyu/rclone/ftp"
	_ "github.com/miseyu/rclone/googlecloudstorage"
	_ "github.com/miseyu/rclone/http"
	_ "github.com/miseyu/rclone/hubic"
	_ "github.com/miseyu/rclone/local"
	_ "github.com/miseyu/rclone/onedrive"
	_ "github.com/miseyu/rclone/pcloud"
	_ "github.com/miseyu/rclone/qingstor"
	_ "github.com/miseyu/rclone/s3"
	_ "github.com/miseyu/rclone/sftp"
	_ "github.com/miseyu/rclone/swift"
	_ "github.com/miseyu/rclone/webdav"
	_ "github.com/miseyu/rclone/yandex"
)
