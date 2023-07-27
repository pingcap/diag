
QN_ACCESS_KET_ID = "qn_access_key"
QN_SECRET_KEY_ID = "qiniu_secret_key"
IMAGE_PATH = "hub.pingcap.net/clinic/diag"
DIAG_DESC = "Clinic client for data collection and quick health check"

// default tag: nightly
if("${RELEASE_VER}" == "") {
	env.RELEASE_TAG = "nightly"
} else {
	env.RELEASE_TAG = "${RELEASE_VER}".toLowerCase()

	//  nightly version does not need a special prefix/suffix
	if( !"${RELEASE_TAG}".contains("nightly")) {
		RELEASE_TAG = "nightly"
	}
} 

// default branche: master
if("${GIT_REF}" == "") {
	env.BUILD_BRANCH = "master"
} else {
	env.BUILD_BRANCH = "${GIT_REF}"
}

// default git url: https://github.com/pingcap/diag.git
if("${GIT_URL}" == "") {
	env.BUILD_URL = "https://github.com/pingcap/diag.git"
} else {
	env.BUILD_URL = "${GIT_URL}"
}

// default tiup mirror url: https://tiup-mirrors.pingcap.com
if("${TIUP_MIRROR}" == "") {
	env.TIUP_MIRROR =  "https://tiup-mirrors.pingcap.com"
} else {
	env.TIUP_MIRROR = "${TIUP_MIRROR}"
}


def run_with_pod(Closure body) {
	label = "clinic-diag"
    def cloud = "kubernetes"
    def namespace = "jenkins-tidb"
    def pod_go_docker_image = "hub.pingcap.net/clinic/centos7_golang-1.18:latest" 
    podTemplate(label: label,
            cloud: cloud,
            namespace: namespace,
            idleMinutes: 0,
            containers: [
                    containerTemplate(
                            name: 'golang', alwaysPullImage: false,
                            image: "${pod_go_docker_image}", ttyEnabled: true,
                            resourceRequestCpu: '2000m', resourceRequestMemory: '4Gi',
                            command: '/bin/sh -c', args: 'cat',
                            envVars: [containerEnvVar(key: 'GOPATH', value: '/go')], 
                    ),
					containerTemplate(
						name: 'docker', 
						image: 'docker:18.09-dind',
						ttyEnabled: true, 
						alwaysPullImage: true, 
						privileged: true,
						command: 'dockerd --host=unix:///var/run/docker.sock --host=tcp://0.0.0.0:2375 --storage-driver=overlay'
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

def checkout_scm(BUILD_URL, BUILD_BRANCH) {
	deleteDir()

	specStr = '+refs/heads/*:refs/remotes/origin/* +refs/pull/*:refs/remotes/origin/pr/*'
	try {
		checkout scm: [
				$class: 'GitSCM',
				branches: [[name: "${BUILD_BRANCH}"]],
				userRemoteConfigs: [[
						credentialsId: 'github-sre-bot',
						refspec: specStr,
						url: "${BUILD_URL}",
				]]
		]
	} catch (info) {
		retry(3) {
			echo "checkout failed, retry.."
			sleep 10
			checkout scm: [
					$class: 'GitSCM',
					branches: [[name: "${BUILD_BRANCH}"]],
					userRemoteConfigs: [[
							credentialsId: 'github-sre-bot',
							refspec: specStr,
							url: "${BUILD_URL}",
					]]
			]
		}
	}
}


def push_image(IMAGE_PATH, RELEASE_TAG) {
	// registry list
	// registryName : [ project,  credentialsId]
	def Registrys  = [
		"hub.pingcap.net": ["clinic","harbor-pingcap"], 
		"docker.io": ["pingcap", "dockerhub"],
		"registry.cn-beijing.aliyuncs.com": ["tidb", "ACR_TIDB_ACCOUNT"],
	]

	// tag && push image
	for ( registry in Registrys.keySet() ) {

		registry_address = "https://${registry}"
		// get project name
		project = Registrys.get(registry)[0]
		registry_image_path = "${registry}/${project}/diag"
			
		// docker.io 
		if ( "${registry}"== "docker.io" ) {
			registry_address = ""
			registry_image_path = "${project}/diag"
		}

		docker.withRegistry(registry_address, Registrys.get(registry)[1]) {

			echo "Login Registry:${registry}    Project: ${project}"
			sh("docker tag ${IMAGE_PATH}:${RELEASE_TAG} ${registry_image_path}:${RELEASE_TAG}")
			sh("docker push ${registry_image_path}:${RELEASE_TAG}")
			
			// update tag lastets
			if( !"${RELEASE_TAG}".contains("nightly")) {	
				sh("docker tag ${IMAGE_PATH}:${RELEASE_TAG} ${registry_image_path}:latest")
				sh("docker push ${registry_image_path}:latest")
			}
		}
	}
}


def publish_chart(RELEASE_TAG) {

	chartPrefixName = "diag-" + "${env.RELEASE_TAG}"
	
	echo "	Generate diag Chart"
	sh """
		sed -i "s/version:.*/version: ${RELEASE_TAG}/g"  k8s/chart/diag/Chart.yaml
		sed -i "s/appVersion:.*/appVersion: ${RELEASE_TAG}/g" k8s/chart/diag/Chart.yaml
		sed -i "s/imageTag:.*/imageTag: ${RELEASE_TAG}/g" k8s/chart/diag/values.yaml		
	"""
	sh "cat k8s/chart/diag/Chart.yaml"
	sh "cat k8s/chart/diag/values.yaml"

	echo "	Package Chart file"
	sh "tar -zcf ${chartPrefixName}.tgz -C k8s/chart diag"
	sh "sha256sum ${chartPrefixName}.tgz > ${chartPrefixName}.sha256"
	sh "ls -l | grep ${chartPrefixName}"
	
	if( !"${RELEASE_TAG}".contains("nightly")) {
		upload_chart("${chartPrefixName}")
	}
}

def upload_chart(chartPrefixName) {
	echo "Upload Chart index.yaml"
	
	// get qinui key
	withCredentials(
		[
			string(
				credentialsId: "${QN_ACCESS_KET_ID}", 
				variable: 'QN_access_key'), 
			string(
				credentialsId: "${QN_SECRET_KEY_ID}", 
				variable: 'Qiniu_secret_key'
				)
		]
	) {
		ACCESS_KEY = QN_access_key
		SECRET_KEY = Qiniu_secret_key
		}

	echo "	Install helm"
	sh """
		wget https://get.helm.sh/helm-v2.14.1-linux-amd64.tar.gz
		tar -zxvf helm-v2.14.1-linux-amd64.tar.gz
		mv linux-amd64/helm ./
		chmod +x ./helm
	"""

	echo "	merge index.yaml"
	sh "curl http://charts.pingcap.org/index.yaml -o index.yaml"
	sh "cat index.yaml"	
	sh "./helm repo index . --url http://charts.pingcap.org/ --merge index.yaml"	
	sh "cat index.yaml"	

	echo "	download Qinui upload tool"
	sh """
		curl https://raw.githubusercontent.com/pingcap/docs-cn/a4db3fc5171ed8e4e705fb34552126a302d29c94/scripts/upload.py -o upload.py
		sed -i 's%https://download.pingcap.org%https://charts.pingcap.org%g' upload.py
		sed -i 's/python3/python/g' upload.py
		chmod +x upload.py
	"""

	echo "	upload chart to QiNui"
	sh """
		set +x
		export QINIU_ACCESS_KEY="${ACCESS_KEY}"
		export QINIU_SECRET_KEY="${SECRET_KEY}"
		export QINIU_BUCKET_NAME="charts"
		set -x
		python3 upload.py ${chartPrefixName}.tgz ${chartPrefixName}.tgz
		python3 upload.py ${chartPrefixName}.sha256 ${chartPrefixName}.sha256
		python3 upload.py index.yaml index.yaml
	"""
}

def publish_diag_cli(RELEASE_TAG, TIUP_MIRROR) {
	
	OS_Arch = [
		"linux": ["amd64", "arm64"],
		"darwin" : ["amd64", "arm64"]
	]

	// set TiUP mirror address
	sh "tiup mirror set ${TIUP_MIRROR}"

	// set tiup private key
	sh "mkdir -p /home/jenkins/.tiup/keys"
	sh "ls -l /home/jenkins/.tiup/keys"
	withCredentials([file(credentialsId: "tiup_private", variable: "tmp_private")]) {
		sh "mv ${tmp_private}  /home/jenkins/.tiup/keys/private.json"
	}
	sh "ls -l /home/jenkins/.tiup/keys"

	currDir = pwd()
	

	// build and publish
	for ( os in OS_Arch.keySet() ) {
		for ( arch in OS_Arch.get(os) ) {
			// init 
			sh "pwd && rm -rf bin/* && mkdir -p bin"

			withCredentials([file(credentialsId: "diag_private", variable: "tmp_private")]) {
				sh "mv ${tmp_private} ${currDir}/bin/pingcap.crt"
			}

			echo "	build diag-${os}/${arch}"
			sh """BUILD_FLAGS="-trimpath -mod=readonly -modcacherw -buildvcs=false" GOOS="${os}" GOARCH="${arch}"  make build"""
			// check binary
			sh "pwd && ls -l bin"
		
			if( "${RELEASE_TAG}".contains("nightly")) {
				time = sh(returnStdout: true, script: "date '+%Y%m%d'").trim()
				prefix = sh(returnStdout: true, script: "git describe --tags --dirty --always").trim()
				//  nightly version does not require a special prefix/suffix
				RELEASE_TAG = "${prefix}-nightly-${time}"
			} 

			// publish
			sh """
				tiup package `ls bin` -C bin --name=diag --release=${RELEASE_TAG} --entry=diag --os=${os} --arch=${arch} --desc="${DIAG_DESC}"
				tiup mirror publish diag ${RELEASE_TAG} package/diag-${RELEASE_TAG}-${os}-${arch}.tar.gz diag --arch ${arch} --os ${os} --desc="${DIAG_DESC}"
			"""
		}

	}

}

run_with_pod { 
	
	container("golang") {

		stage("Checkout") {

			// check go version
			sh "whoami && go version"
				
			echo "Start checkout"
			echo "BUILD_URL: ${env.BUILD_URL}"
			echo "BUILD_BRANCH: ${env.BUILD_BRANCH}"
			echo "RELEASE_TAG: ${env.RELEASE_TAG}"

			checkout_scm("${env.BUILD_URL}", "${env.BUILD_BRANCH}")

			// check code
			sh "pwd && ls -l"
			echo "Checkout success"
		}

		stage("Check and Test") {
			// check code
			sh "make check"
			sh "make test"
				
		}

		stage('Publish diag cli binary') {

			echo "Start publish diag cli"
			echo "RELEASE_TAG: ${env.RELEASE_TAG}"
			echo "TIUP_MIRROR: ${env.TIUP_MIRROR}"


			publish_diag_cli("${env.RELEASE_TAG}", "${env.TIUP_MIRROR}")

			sh "pwd && ls -l package"
			sh "cat package/tiup-component-diag.index"
			sh "cat package/tiup-manifest.index"

			echo "Publish diag cli success"
		}
/*
		stage("Build k8s binary") {
			// build 
			sh """BUILD_FLAGS="-trimpath -mod=readonly -modcacherw -buildvcs=false" make k8s"""
		}
*/
	}	
/*
	container("docker") {
		
		stage("Build and Push docker image") {

			echo "Start push image"
			echo "LOCAL_IMAGE_PATH: ${IMAGE_PATH}"
			echo "RELEASE_TAG: ${env.RELEASE_TAG}"

			// check k8s images
			sh "cp configs/info.toml k8s/images/diag/bin/"
			docker.withRegistry("", "dockerhub") {
			    sh("pwd && ls -l k8s/images")
			    sh("docker build --tag ${IMAGE_PATH}:${env.RELEASE_TAG} -f k8s/images/diag/Dockerfile k8s/images/diag")
			    sh("docker image list")
			}

			push_image("${IMAGE_PATH}", "${env.RELEASE_TAG}")

			sh "docker image list"
			echo "Push image success" 
		}
	}

	container("golang") {

		stage('Publish diag chart') {

			// check charts
			sh "pwd && ls -l k8s/chart/diag"

			echo "Release diag:${env.RELEASE_TAG} chart"

			echo "QN_ACCESS_KET_ID: ${QN_ACCESS_KET_ID}"
			echo "QN_SECRET_KEY_ID: ${QN_SECRET_KEY_ID}"

			publish_chart("${env.RELEASE_TAG}")
			
			echo "Publish chart success"
		}
	}
*/
}
