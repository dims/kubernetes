# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ARG BASEIMAGE
FROM $BASEIMAGE

CROSS_BUILD_COPY qemu-QEMUARCH-static /usr/bin/

RUN apt-get update && apt-get install -y wget time

# Use TensorFlow 2.15 which is the last version with tf.estimator support.
# TensorFlow 2.16+ removed tf.estimator which is required by the models v1.9.0 code.
RUN pip install tensorflow==2.15.0

# Copy the TF1->TF2 compatibility shim and patch script
COPY tf_compat.py /tf_compat.py
COPY patch_imports.py /patch_imports.py

# Use models v1.9.0 which contains the wide_deep implementation.
# The wide_deep directory was removed in later versions of tensorflow/models.
RUN wget https://github.com/tensorflow/models/archive/v1.9.0.tar.gz \
    && tar xzf v1.9.0.tar.gz \
    && rm -f v1.9.0.tar.gz

# Patch all Python files to use the TF1 compatibility shim.
# The models v1.9.0 code uses TF1 APIs (tf.app, tf.gfile, tf.VERSION, etc.)
# that were moved to tf.compat.v1 in TF2.
RUN python3 /patch_imports.py \
    "/models-1.9.0/official/wide_deep/*.py" \
    "/models-1.9.0/official/utils/*.py" \
    "/models-1.9.0/official/utils/flags/*.py" \
    "/models-1.9.0/official/utils/logs/*.py"

RUN python /models-1.9.0/official/wide_deep/data_download.py

WORKDIR $HOME/models-1.9.0/official/wide_deep
ENV PYTHONPATH=$PYTHONPATH:$HOME/models-1.9.0

ENTRYPOINT ["python", "./wide_deep.py"]
