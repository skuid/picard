import groovy.json.JsonSlurperClassic;

node("generic"){

    stage("Fetch artifacts"){
        step([
            $class: 'CopyArtifact',
            filter: 'artifacts/docker-image.txt, artifacts/revision.txt',
            fingerprintArtifacts: true,
            projectName: "builds/warden/${env.BRANCH}",
        ]);
    }

    def revision = readFile("artifacts/revision.txt").trim();
    def imageName = readFile("artifacts/docker-image.txt").trim();

    echo """
Deploying git commit ${revision} to ${env.DEPLOY_TARGET}...
"""

    stage("Deploy Image"){
        sh "kubectl set image -n platform deployment/warden-dal warden=${imageName}";
    }

}
