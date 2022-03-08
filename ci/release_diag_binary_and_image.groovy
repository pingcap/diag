def run_with_pod(Closure body) {
    def cloud = "kubernetes"
    def namespace = "jenkins-tidb"
    def jnlp_docker_image = "jenkins/inbound-agent:4.3-4"
    podTemplate(label: label,
            cloud: cloud,
            namespace: namespace,
            idleMinutes: 0,
            containers: [
                    containerTemplate(
                            name: 'golang', alwaysPullImage: false,
                            image: "${pod_go_docker_image}", ttyEnabled: true,
                            resourceRequestCpu: '6000m', resourceRequestMemory: '16Gi',
                            command: '/bin/sh -c', args: 'cat',
                            envVars: [containerEnvVar(key: 'GOPATH', value: '/go')], 
                    ),
            ],
           
    ) {
        node(label) {
            println "debug command:\nkubectl -n ${namespace} exec -ti ${NODE_NAME} bash"
            timeout(time: 60, unit: 'MINUTES') {
               body() 
            }
        }
    }
}


run_with_pod { 
    d WORKSPACE = pwd()

    stage("Checkout") {
            container("golang") {
                sh "whoami && go version"
            }
            // update code
            dir("${WORKSPACE}") {
				deleteDir()
				
				try {
					checkout changelog: false, poll: false, 
					scm: [
						$class: 'GitSCM', 
						branches: [[name: "${BUILD_BRANCH}"]], 
						url: "${BUILD_URL}",
						userRemoteConfigs: [[credentialsId: 'github-sre-bot-ssh', refspec: specStr, url: "${BUILD_URL}"]]]
				}   catch (info) {
						retry(2) {
							echo "checkout failed, retry.."
							sleep 5
							if (sh(returnStatus: true, script: '[ -d .git ] && [ -f Makefile ] && git rev-parse --git-dir > /dev/null 2>&1') != 0) {
								deleteDir()
							}
							checkout changelog: false, poll: false, scm: [$class: 'GitSCM', branches: [[name: 'master']], doGenerateSubmoduleConfigurations: false, extensions: [[$class: 'PruneStaleBranch'], [$class: 'CleanBeforeCheckout'], [$class: 'CloneOption', timeout: 10]], submoduleCfg: [], userRemoteConfigs: [[credentialsId: 'github-sre-bot-ssh', refspec: specStr, url: 'git@github.com/srstack/diag.git']]]
						}
				}

			sh "pwd $$ ls -l"
        }

    


}


def call(BUILD_BRANCH, RELEASE_TAG, CREDENTIALS_ID, CHART_ITEMS) {
	def GITHASH
	def ACCESS_KEY
	def SECRET_KEY
	def UCLOUD_OSS_URL = "http://pingcap-dev.hk.ufileos.com"

	catchError {
		node('delivery') {
			container("delivery") {
				def WORKSPACE = pwd()
				withCredentials([string(credentialsId: "${env.QN_ACCESS_KET_ID}", variable: 'QN_access_key'), string(credentialsId: "${env.QN_SECRET_KEY_ID}", variable: 'Qiniu_secret_key')]) {
					ACCESS_KEY = QN_access_key
					SECRET_KEY = Qiniu_secret_key
				}

				sh "chown -R jenkins:jenkins ./"
				deleteDir()

				dir("${WORKSPACE}/diag") {
					stage('Download diag binary'){
						// "http://pingcap-dev.hk.ufileos.com/refs/pingcap
						GITHASH = sh(returnStdout: true, script: "curl ${UCLOUD_OSS_URL}/refs/pingcap/operator/${BUILD_BRANCH}/centos7/sha1").trim()
						sh "curl ${UCLOUD_OSS_URL}/builds/pingcap/operator/${GITHASH}/centos7/diag.tar.gz | tar xz"
					}

                    def images = ["diag"]
                    images.each {
                        stage("Build and push ${it} image") {
                            withDockerServer([uri: "${env.DOCKER_HOST}"]) {
                                docker.build("pingcap/${it}:${RELEASE_TAG}", "images/${it}").push()
                                withDockerRegistry([url: "https://registry.cn-beijing.aliyuncs.com", credentialsId: "ACR_TIDB_ACCOUNT"]) {
                                    sh """
                                    docker tag pingcap/${it}:${RELEASE_TAG} registry.cn-beijing.aliyuncs.com/tidb/${it}:${RELEASE_TAG}
                                    docker push registry.cn-beijing.aliyuncs.com/tidb/${it}:${RELEASE_TAG}
                                    """
                                }
                            }
                        }
                    }

					stage('Publish charts to charts.pingcap.org') {
						ansiColor('xterm') {
						sh """
						set +x
						export QINIU_ACCESS_KEY="${ACCESS_KEY}"
						export QINIU_SECRET_KEY="${SECRET_KEY}"
						export QINIU_BUCKET_NAME="charts"
						set -x
						curl https://raw.githubusercontent.com/pingcap/docs-cn/a4db3fc5171ed8e4e705fb34552126a302d29c94/scripts/upload.py -o upload.py
						sed -i 's%https://download.pingcap.org%https://charts.pingcap.org%g' upload.py
						sed -i 's/python3/python/g' upload.py
						chmod +x upload.py
						for chartItem in ${CHART_ITEMS}
						do
							chartPrefixName=\$chartItem-${RELEASE_TAG}
							echo "======= release \$chartItem chart ======"
							sed -i "s/version:.*/version: ${RELEASE_TAG}/g" charts/\$chartItem/Chart.yaml
							sed -i "s/appVersion:.*/appVersion: ${RELEASE_TAG}/g" charts/\$chartItem/Chart.yaml
                            # update image tag to current release
                            sed -r -i "s#pingcap/(tidb-operator|tidb-backup-manager):.*#pingcap/\\1:${RELEASE_TAG}#g" charts/\$chartItem/values.yaml
							tar -zcf \${chartPrefixName}.tgz -C charts \$chartItem
							sha256sum \${chartPrefixName}.tgz > \${chartPrefixName}.sha256
							./upload.py \${chartPrefixName}.tgz \${chartPrefixName}.tgz
							./upload.py \${chartPrefixName}.sha256 \${chartPrefixName}.sha256
						done
						# Generate index.yaml for helm repo if the version is not "latest" (not a valid semantic version)
                        if [ "${RELEASE_TAG}" != "latest" -a "${RELEASE_TAG}" != "nightly" ]; then
                            wget https://get.helm.sh/helm-v2.14.1-linux-amd64.tar.gz
                            tar -zxvf helm-v2.14.1-linux-amd64.tar.gz
                            mv linux-amd64/helm /usr/local/bin/helm
                            chmod +x /usr/local/bin/helm
                            #ls
                            curl http://charts.pingcap.org/index.yaml -o index.yaml
                            cat index.yaml
                            helm repo index . --url http://charts.pingcap.org/ --merge index.yaml
                            cat index.yaml
                            ./upload.py index.yaml index.yaml
                        else
                            echo "info: RELEASE_TAG is ${RELEASE_TAG}, skip adding it into chart index file"
                        fi
						"""
						}
					}
				}
			}
		}
		currentBuild.result = "SUCCESS"
	}

	stage('Summary') {
		echo("echo summary info ########")
		def DURATION = ((System.currentTimeMillis() - currentBuild.startTimeInMillis) / 1000 / 60).setScale(2, BigDecimal.ROUND_HALF_UP)
		def slackmsg = "[${env.JOB_NAME.replaceAll('%2F','/')}-${env.BUILD_NUMBER}] `${currentBuild.result}`" + "\n" +
		"Elapsed Time: `${DURATION}` Mins" + "\n" +
		"tidb-operator Branch: `${BUILD_BRANCH}`, Githash: `${GITHASH.take(7)}`" + "\n" +
		"Display URL:" + "\n" +
		"${env.RUN_DISPLAY_URL}"

		if(currentBuild.result != "SUCCESS"){
			slackSend channel: '#cloud_jenkins', color: 'danger', teamDomain: 'pingcap', tokenCredentialId: 'slack-pingcap-token', message: "${slackmsg}"
			return
		}

		slackmsg = "${slackmsg}" + "\n" +
		"clinic diag Docker Image: `pingcap/diag:${RELEASE_TAG}`" + "\n" +
		"clinic diag Docker Image: `uhub.ucloud.cn/pingcap/diag:${RELEASE_TAG}`"


		for(String chartItem : CHART_ITEMS.split(' ')){
			slackmsg = "${slackmsg}" + "\n" +
			"${chartItem} charts Download URL: http://charts.pingcap.org/${chartItem}-${RELEASE_TAG}.tgz"
		}
		slackmsg = "${slackmsg}" + "\n" +
		"charts index Download URL: http://charts.pingcap.org/index.yaml"

		slackSend channel: '#cloud_jenkins', color: 'good', teamDomain: 'pingcap', tokenCredentialId: 'slack-pingcap-token', message: "${slackmsg}"
	}
}

return this