@Library('jenkins.shared.library') _

pipeline {
  agent {
    label 'ubuntu_docker_label'
  }
  environment {
    // signDockerImage() (jenkins.shared.library) reads env.HARBOR_SERVER, not
    // REGISTRY -- without this override it silently falls back to the
    // Jenkins-wide global default (core-harbor-prod.sdp.infoblox.com), which
    // this chart's values.yaml and the cluster never pull from.
    HARBOR_SERVER = "harbor.services.sdp.infoblox.com"
    VERSION = sh(script: "git describe --always --long --tags", returnStdout: true).trim()
    TAG = "${env.VERSION}-j${env.BUILD_NUMBER}"
  }
  stages {
    stage("Build Image") {
      steps {
        sh 'docker build . -t tags-operator:$TAG'
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
