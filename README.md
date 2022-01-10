# 提前安装好FFmpeg的开发库
```bash
sudo apt install -y libavdevice-dev libavfilter-dev libswscale-dev libavcodec-dev libavformat-dev libswresample-dev libavutil-dev
```
# 改动 20220107 

- https://github.com/giorgisio/goav 没人维护了，有很多问题，很多人反映不能用了。原因是：ffmpeg 在持续变化，很多api 过时了。把依赖改到了[github.com/charlestamz/goav](https://github.com/charlestamz/goav)，这个里面去掉了一些过时得api。
- 在plugin里面用不过时得方法改了一下转码得代码

# 主要功能
- 提供音频转码从aac->pcma、pcmu或者反过来
