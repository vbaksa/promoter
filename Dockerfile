FROM library/centos:latest
RUN yum install golang -y && yum clean all
ADD . /tmp/src/github.com/vbaksa/promoter/
# Temporal hack
#ADD . /usr/lib/golang/src/github.com/vbaksa/promoter/
RUN cd /tmp/src/github.com/vbaksa/promoter/ && chmod 755 ./build.sh && ./build.sh
CMD ["/opt/promoter/run.sh"]


