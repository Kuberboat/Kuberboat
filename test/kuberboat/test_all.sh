#!/bin/bash

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )

declare -i exit_code
exit_code=0

bash $parent_path/test_node.sh
if [ $? -ne 0 ]
    then exit_code=1
fi
bash $parent_path/test_pod.sh
if [ $? -ne 0 ]
    then exit_code=1
fi

bash $parent_path/test_service.sh
if [ $? -ne 0 ]
    then exit_code=1
fi

bash $parent_path/test_deployment.sh
if [ $? -ne 0 ]
    then exit_code=1
fi

bash $parent_path/test_recover.sh
if [ $? -ne 0 ]
    then exit_code=1
fi

exit $exit_code
