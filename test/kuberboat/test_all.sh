#!/bin/bash

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )

bash $parent_path/test_node.sh
bash $parent_path/test_pod.sh
bash $parent_path/test_service.sh
bash $parent_path/test_deployment.sh
