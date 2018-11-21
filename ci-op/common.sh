export CI_OP="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
export GOPATH=${GOPATH:-~/go}
export PATH=$PATH:$GOPATH/bin
export PROVIDER_OS=${PROVIDER_OS:-ubuntu}
# Can we assume the existing CAPO src dir is in the go/src path?
export CAPO_SRC=$1

if [ !-d $CAPO_SRC ];
then
    export CAPO_SRC=`realpath $CI_OP/..`
fi
