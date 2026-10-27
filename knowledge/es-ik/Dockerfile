FROM docker.elastic.co/elasticsearch/elasticsearch:8.15.3

# The IK and Elasticsearch versions must stay exactly aligned.
RUN bin/elasticsearch-plugin install --batch \
    https://get.infini.cloud/elasticsearch/analysis-ik/8.15.3

# ES 8.x 中 IK 从 ES 的 config/analysis-ik/ 目录读取配置与扩展词典（不是插件目录）。
# 项目自有水务词典 + 注册配置，保证 elasticsearch 用户可读（否则词典静默不生效）。
COPY --chown=elasticsearch:root custom.dic /usr/share/elasticsearch/config/analysis-ik/custom.dic
COPY --chown=elasticsearch:root IKAnalyzer.cfg.xml /usr/share/elasticsearch/config/analysis-ik/IKAnalyzer.cfg.xml
