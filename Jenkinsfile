@Library('jenkins.shared.library') _

pipeline {
  agent {
    label 'ubuntu_docker_label'
  }
  environment {
    REGISTRY = "core-harbor-prod.sdp.infoblox.com"
    VERSION = sh(script: "git describe --always --long --tags", returnStdout: true).trim()
    TAG = "${env.VERSION}-j${env.BUILD_NUMBER}"
  }
  stages {
    stage("Build Image") {
      steps {
        // The Dockerfile uses RUN --mount (upstream image check), which needs BuildKit.
        // The Docker 20.10 agents ship it but keep it off by default.
        sh 'DOCKER_BUILDKIT=1 docker build . -t tags-operator:$TAG'
      }
    }
    stage("Push Image") {
      when {
        anyOf {
          branch 'main'
          branch 'jenkinsfile'
        }
      }
      steps {
        script {
          signDockerImage('tags-operator', env.TAG, 'infoblox')
        }
      }
    }
  }
  post {
    success {
       finalizeBuild()
    }
  }
}
