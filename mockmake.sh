#!/usr/bin/env bash
mkdir -p mocks
echo "all: _all"
list=${1:-mock_list.txt}
all=""
for l in $(cat ${list});
do
    dir=$(dirname ${l})
    base=$(basename ${l})
    mockbase="mocks"
    if [[ -e "${l}" ]];then
       file="${l}"
    elif [[ -e "./vendor/${l}" ]]; then
       file="./vendor/${l}"
    else
        file=${l}
    fi
    glob=$(ls ${file})
    if [ ! ${?} = 0 ];then
        exit $?
    fi
    for f in ${glob};
    do        
        if [[ ! -e ${f} ]];then
            echo "not found: ${f}"
            exit 1
        fi
        if [[ ${f} =~ _test.go ]];then
            continue
        fi
        dir=$(dirname ${l})
        base=$(basename ${l})
        task="${mockbase}/${dir}/mock_${base}"
        mkdir -p ${mockbase}
        all+=" ${task}"
        echo -e "${task}: ${file}"
        echo -e "\tmkdir -p ${mockbase}/${dir}"
        echo -e "\tmockgen -source ${file} > ${mockbase}/${dir}/mock_${base}"
    done
done
echo -e "_all: ${all}"
echo -e "clean:"
echo -e "\trm -rf mocks/*"
