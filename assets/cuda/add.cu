#include <cuda.h>
#include <cuda_runtime.h>
#include <stdio.h>

#define BLOCK_NUM 8    //块数量
#define THREAD_NUM 64  // 每个块中的线程数
#define R_SIZE BLOCK_NUM *THREAD_NUM
#define M_SIZE R_SIZE *R_SIZE

__global__ void mat_add(int *mat1, int *mat2, int *result) {
  const int bid = blockIdx.x;
  const int tid = threadIdx.x;
  // 每个线程计算一行
  const int row = bid * THREAD_NUM + tid;
  for (int i = 0; i < R_SIZE; i++) {
    int index = row * R_SIZE + i;
    result[index] = mat1[index] * mat2[index];
  }
}

int main(int argc, char *argv[]) {
  int *mat1, *mat2, *result;
  int *g_mat1, *g_mat2, *g_mat_result;

  // 用一位数组表示二维矩阵
  mat1 = (int *)malloc(M_SIZE * sizeof(int));
  mat2 = (int *)malloc(M_SIZE * sizeof(int));
  result = (int *)malloc(M_SIZE * sizeof(int));

  // initialize
  for (int i = 0; i < M_SIZE; i++) {
    mat1[i] = rand() / 1000000;
    mat2[i] = rand() / 1000000;
    result[i] = 0;
  }

  cudaMalloc((void **)&g_mat1, sizeof(int) * M_SIZE);
  cudaMalloc((void **)&g_mat2, sizeof(int) * M_SIZE);
  cudaMalloc((void **)&g_mat_result, sizeof(int) * M_SIZE);

  cudaMemcpy(g_mat1, mat1, sizeof(int) * M_SIZE, cudaMemcpyHostToDevice);
  cudaMemcpy(g_mat2, mat2, sizeof(int) * M_SIZE, cudaMemcpyHostToDevice);

  mat_add<<<BLOCK_NUM, THREAD_NUM>>>(g_mat1, g_mat2, g_mat_result);

  cudaMemcpy(result, g_mat_result, sizeof(int) * M_SIZE,
             cudaMemcpyDeviceToHost);
  printf("mat1[0][0] is %d\tmat2[0][0] is %d\treault[0][0] is %d\n", mat1[0],
         mat2[0], result[0]);
  free(mat1);
  free(mat2);
  free(result);
}