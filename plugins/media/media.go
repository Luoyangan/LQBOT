// Package media demonstrates sending rich media messages (image/video/voice/file).
// URL must be pre-registered on QQ Open Platform (开发设置→消息URL配置).
package media

import (
	"github.com/Luoyangan/LQBOT/internal/contract"
)

// Register adds the image and video commands.
// Demonstrates: SendImage, SendGroupRichMedia, SendC2CRichMedia, scene-aware API.
// NOTE: Replace example URLs with your own pre-registered URLs.
func Register(r contract.CommandRegister, api contract.QQAPI) {
	// /image — sends an image
	r.Register(contract.Command{
		Name:        "image",
		Description: "发送一张示例图片",
		Usage:       "image",
		Handler: func(ctx contract.CommandContext) error {
			imageURL := "https://wd.lilei007.cn/Luoyangan/1.png"

			switch ctx.Scene() {
			case contract.SceneGroup:
				return api.SendGroupRichMedia(ctx.GroupID(), &contract.RichMedia{
					FileType: 1,
					URL:      imageURL,
					MsgID:    ctx.MessageID(), // 被动回复，绕过主动消息限制
				})
			case contract.SceneC2C:
				return api.SendC2CRichMedia(ctx.AuthorID(), &contract.RichMedia{
					FileType: 1,
					URL:      imageURL,
					MsgID:    ctx.MessageID(),
				})
			default:
				return api.ReplyImage(ctx.ChannelID(), ctx.MessageID(), imageURL)
			}
		},
	})

	// /video — sends a video (group/C2C only via RichMedia)
	r.Register(contract.Command{
		Name:        "video",
		Description: "发送一个示例视频",
		Usage:       "video",
		Handler: func(ctx contract.CommandContext) error {
			videoURL := "https://wd.lilei007.cn/Luoyangan/1.mp4"

			switch ctx.Scene() {
			case contract.SceneGroup:
				return api.SendGroupRichMedia(ctx.GroupID(), &contract.RichMedia{
					FileType: 2, // video
					URL:      videoURL,
					Content:  "看这个视频",
					MsgID:    ctx.MessageID(),
				})
			case contract.SceneC2C:
				return api.SendC2CRichMedia(ctx.AuthorID(), &contract.RichMedia{
					FileType: 2,
					URL:      videoURL,
					MsgID:    ctx.MessageID(),
				})
			default:
				return ctx.Reply("视频消息目前仅支持群聊和 C2C 场景。")
			}
		},
	})

	// /voice — sends a voice message (group/C2C only)
	r.Register(contract.Command{
		Name:        "voice",
		Description: "发送一个示例语音",
		Usage:       "voice",
		Handler: func(ctx contract.CommandContext) error {
			voiceURL := "https://wd.lilei007.cn/Luoyangan/1.mp3"

			switch ctx.Scene() {
			case contract.SceneGroup:
				return api.SendGroupRichMedia(ctx.GroupID(), &contract.RichMedia{
					FileType: 3, // voice
					URL:      voiceURL,
					Content:  "听这段语音",
					MsgID:    ctx.MessageID(),
				})
			case contract.SceneC2C:
				return api.SendC2CRichMedia(ctx.AuthorID(), &contract.RichMedia{
					FileType: 3,
					URL:      voiceURL,
					MsgID:    ctx.MessageID(),
				})
			default:
				return ctx.Reply("语音消息目前仅支持群聊和 C2C 场景。")
			}
		},
	})

	// /file — sends a file (group/C2C only)
	r.Register(contract.Command{
		Name:        "file",
		Description: "发送一个示例文件",
		Usage:       "file",
		Handler: func(ctx contract.CommandContext) error {
			fileURL := "https://wd.lilei007.cn/Luoyangan/1.zip"

			switch ctx.Scene() {
			case contract.SceneGroup:
				return api.SendGroupRichMedia(ctx.GroupID(), &contract.RichMedia{
					FileType: 4, // file
					URL:      fileURL,
					Content:  "下载这个文件",
					MsgID:    ctx.MessageID(),
				})
			case contract.SceneC2C:
				return api.SendC2CRichMedia(ctx.AuthorID(), &contract.RichMedia{
					FileType: 4,
					URL:      fileURL,
					MsgID:    ctx.MessageID(),
				})
			default:
				return ctx.Reply("文件消息目前仅支持群聊和 C2C 场景。")
			}
		},
	})
}
