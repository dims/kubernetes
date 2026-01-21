"""TensorFlow 1.x to 2.x compatibility shim for wide_deep model.

This module provides backward compatibility for TF1 APIs that were removed
or renamed in TensorFlow 2.x. The models v1.9.0 code uses TF1-style APIs
that need to be shimmed for TF2 compatibility.
"""
import tensorflow as tf


class GFileCompat:
    """Wrapper to provide TF1-style gfile API on top of TF2 tf.io.gfile.

    TF1 used capitalized method names (Exists, Open, MkDir, etc.) while
    TF2 uses lowercase (exists, GFile, makedirs, etc.). This wrapper
    translates the old names to the new ones.
    """

    def __getattr__(self, name):
        tf1_to_tf2 = {
            "Exists": "exists",
            "Open": "GFile",
            "MkDir": "makedirs",
            "MakeDirs": "makedirs",
            "Remove": "remove",
            "Rename": "rename",
            "Copy": "copy",
            "Glob": "glob",
            "IsDirectory": "isdir",
            "ListDirectory": "listdir",
            "Walk": "walk",
            "Stat": "stat",
            "DeleteRecursively": "rmtree",
            "FastGFile": "GFile",
        }

        if name in tf1_to_tf2:
            return getattr(tf.io.gfile, tf1_to_tf2[name])
        return getattr(tf.io.gfile, name.lower() if name[0].isupper() else name)


# Install the compatibility shims for TF1 APIs
tf.app = tf.compat.v1.app
tf.gfile = GFileCompat()
tf.logging = tf.compat.v1.logging
tf.flags = tf.compat.v1.flags

# Add ConfigProto and Session (moved to tf.compat.v1 in TF2)
if not hasattr(tf, "ConfigProto"):
    tf.ConfigProto = tf.compat.v1.ConfigProto
if not hasattr(tf, "Session"):
    tf.Session = tf.compat.v1.Session
if not hasattr(tf, "global_variables_initializer"):
    tf.global_variables_initializer = tf.compat.v1.global_variables_initializer

# Add VERSION and GIT_VERSION (renamed to __version__ in TF2)
if not hasattr(tf, "VERSION"):
    tf.VERSION = tf.__version__
if not hasattr(tf, "GIT_VERSION"):
    tf.GIT_VERSION = getattr(tf, "__git_version__", "unknown")

# Add SessionRunHook and other hooks to tf.train from tf.estimator
if hasattr(tf, "estimator"):
    if not hasattr(tf.train, "SessionRunHook"):
        tf.train.SessionRunHook = tf.estimator.SessionRunHook
    if not hasattr(tf.train, "StopAtStepHook"):
        tf.train.StopAtStepHook = tf.estimator.StopAtStepHook
    if not hasattr(tf.train, "CheckpointSaverHook"):
        tf.train.CheckpointSaverHook = tf.estimator.CheckpointSaverHook
    if not hasattr(tf.train, "LoggingTensorHook"):
        tf.train.LoggingTensorHook = tf.estimator.LoggingTensorHook
    if not hasattr(tf.train, "NanTensorHook"):
        tf.train.NanTensorHook = tf.estimator.NanTensorHook
    if not hasattr(tf.train, "SummarySaverHook"):
        tf.train.SummarySaverHook = tf.estimator.SummarySaverHook
    if not hasattr(tf.train, "ProfilerHook"):
        tf.train.ProfilerHook = tf.estimator.ProfilerHook

# Set logging verbosity
tf.compat.v1.logging.set_verbosity(tf.compat.v1.logging.INFO)

print("TF1->TF2 compatibility shim loaded successfully")
