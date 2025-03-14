# Hyperliquid网格交易系统安装指南

为了速度和稳定性，我使用的是[Linode VPS](https://www.linode.com/lp/refer/?r=7df1983e917d3958b68bebdf2b6f030e6e9ecb9c)（个人用了10多年了，一直非常稳定），一个月只需要5美元。

## 安装步骤

1. 安装Linux环境，推荐Ubuntu 24.04 LTS

2. 安装Go环境
   ```
   snap install go --classic
   ```

3. 下载代码
   ```
   git clone https://github.com/0xdamahou/hyperliquid-grid
   ```

4. 下载相关库，并构建程序
   ```
   go mod download
   go build -o hypergrid
   ```

5. 安装PostgreSQL
   ```
   apt install postgresql
   ```

6. 运行网格程序只需要3个文件：`hypergrid`、`config.json`和`grid_config.json`，可以将这三个文件放在一个新的目录下面。

7. 修改config.json文件，主要包括hyperliquid api相关的账号地址`api_key`和私钥`api_secret`，数据库和密码信息`db_connection`，是否启用网页`enable_web`来查看网格运行情况。也可以直接在数据库中查看，只是网页查看更直观，推荐启用。
   ```json
   {
     "api_key": "Your-hyperliquid-account-address",
     "api_secret": "your-hyperliquid-api-secret",
     "db_connection": "user=postgres password=your-postgres-password dbname=hyper host=localhost port=5432 sslmode=disable",
     "enable_web": true,
     "web_password": "用于查看网格运行结果的密码，或者由程序随机生成",
     "web_port": ":80"
   }
   ```

8. 然后在grid_config.json修改要运行的网格，每个网格包含如下信息：
   ```json
   {
     "symbol": "SOL",
     "initial_size": 20,
     "grid_step": 0.017,
     "grid_size": 1,
     "leverage": 5,
     "price_precision": 2,
     "act": "Start",
     "enable": true
   }
   ```

    - `symbol`：交易对信息
    - `initial_size`：最开始仓位，比如开始做多20个SOL，如果做空，请使用负数，比如-20，表示做空20个SOL
    - `grid_step`：网格间距，比如0.017表示1.7%
    - `leverage`：杠杆倍数，以什么样的杠杆倍数下单，建议实际杠杆率不要超过1倍，此处杠杆率可以设置5或者10，以便以后网格交易方便
    - `price_precision`：Price的精度，也就是有几位小数，方便计算不同level的网格的价格
    - `act`：可以支持Start和Stop，如果想运行这个网格，一直保持Start状态就可以，如果想退出这个网格，并让系统自动平仓，请修改成`Stop`
    - `enable`：如果不想系统启用这个网格，请修改成false

9. 正式启动网格程序，可在outlog.log中查看运行记录
   ```
   nohup ./hypergrid &
   ```