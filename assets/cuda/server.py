import os
import time
import sys
from fabric import Connection
from invoke.exceptions import UnexpectedExit

host = 'login.hpc.sjtu.edu.cn'
user = 'stu658'
password = 'rAnx&q1F'

hpc_dir = f'/lustre/home/acct-stu/{user}/cuda-test'
cuda_path = '/src/cuda/cuda.cu'
retry_times = 3
query_times = 50
gap_between_query = 1
gap_if_pending = 30


def main():
    with Connection(host=host, user=user, connect_kwargs={'password': password}) as c:
        job_id = 0
        is_ok = False

        for i in range(retry_times):
            try:
                c.put(cuda_path, remote=hpc_dir)
            except Exception as e:
                print(f'round {i}: {e}')
            else:
                is_ok = True
                break
        if not is_ok:
            print('failed to transfer the cuda file')
            sys.exit(1)

        is_ok = False
        for i in range(retry_times):
            try:
                result = c.run('sbatch cuda-test/cuda.slurm', hide='both')
            except UnexpectedExit as ue:
                print(f'round {i}: {ue}')
            else:
                job_id = int(result.stdout.strip().split(' ')[-1])
                is_ok = True
                break
        if not is_ok:
            print('failed to sbatch slurm script')
            sys.exit(1)

        time.sleep(5)

        is_ok = False
        for i in range(query_times):
            try:
                result = c.run(f'squeue | grep {job_id}', hide='both')
            except UnexpectedExit as ue:
                is_ok = True
                print(f'job {job_id} finished')
                break
            else:
                job_info = list(filter(None, result.stdout.strip().split(' ')))
                job_status = job_info[4]
                if job_status == 'PD':
                    print('job is pending; may need to wait for a long time')
                    time.sleep(gap_if_pending)
                else:
                    print(f'job is {job_status}')
                    time.sleep(gap_between_query)
        if not is_ok:
            print('failed to complete the job')
            sys.exit(1)

        is_ok = False
        for i in range(retry_times):
            try:
                c.get(f'{hpc_dir}/{job_id}.err', f'{job_id}.err')
                c.get(f'{hpc_dir}/{job_id}.out', f'{job_id}.out')
            except FileNotFoundError as fe:
                print(f'round {i}: {fe}')
            else:
                is_ok = True
                break
        if not is_ok:
            print('failed to get job\'s output files')

        if os.path.getsize(f'{job_id}.err') != 0:
            print('job failed! error message is as follows')
            with open(f'{job_id}.err') as f:
                print(f.read())
            sys.exit(1)
        else:
            print('job succeed! output is as follows')
            with open(f'{job_id}.out') as f:
                print(f.read())
            sys.exit(0)


if __name__ == '__main__':
    main()
