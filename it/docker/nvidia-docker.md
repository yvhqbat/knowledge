# GPU环境搭建
参考：
- [https://github.com/NVIDIA/nvidia-docker](https://github.com/NVIDIA/nvidia-docker)
- [Nvidia-docker原理及GPU环境搭建](https://tmcdcgeek.club/2017/11/16/nvidia-docker/)
- [https://hub.docker.com/r/nvidia/cuda/](https://hub.docker.com/r/nvidia/cuda/)


搭建`GPU`应用步骤：
1. 在Host操作系统上安装 `Cuda Dirver`
2. 安装 `Docker Engine`
3. 安装 `Nvidia Docker`
4. 基于基础镜像 `docker pull nvidia/cuda` 部署应用

