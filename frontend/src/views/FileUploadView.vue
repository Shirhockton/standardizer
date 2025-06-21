<template>
  <div class="file-upload-container">
    <input type="file" @change="handleFileChange" class="file-input" />
    <button @click="uploadFile" class="btn btn-primary">上传文件</button>
    <button @click="startScan" class="btn btn-secondary">开始扫描</button>
    <button @click="downloadReport" class="btn btn-success">报告下载</button>
    <div v-if="uploadStatus" class="status-message">{{ uploadStatus }}</div>
    <div v-if="scanStatus" class="status-message">{{ scanStatus }}</div>
  </div>
</template>

<script setup lang="ts">
import { onUnmounted, ref } from 'vue';
import { useAuthStore } from '../store/auth';

const selectedFile = ref<File | null>(null);
const uploadStatus = ref('');
const scanStatus = ref('');
let pollInterval: NodeJS.Timeout | null = null; // 用于存储轮询定时器

const handleFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement;
  if (target.files && target.files.length > 0) {
    selectedFile.value = target.files[0];
  }
};

const authStore = useAuthStore();

const uploadFile = async () => {
  if (!selectedFile.value) {
    uploadStatus.value = '请先选择文件';
    return;
  }

  const formData = new FormData();
  formData.append('file', selectedFile.value);

  try {
    const response = await fetch('api/api/upload', {
      method: 'POST',
      body: formData,
      headers: {
        'Authorization': `${authStore.token}`
      }
    });

    if (response.ok) {
      uploadStatus.value = '文件上传成功';
    } else {
      uploadStatus.value = '文件上传失败';
    }
  } catch (error) {
    uploadStatus.value = '上传过程中出现错误';
  }
};

const startScan = async () => {
  if (!selectedFile.value) {
    scanStatus.value = '请先选择文件';
    return;
  }

  try {
    const response = await fetch('api/api/get-response', {
      method: 'GET',
      headers: {
        'Authorization': `${authStore.token}`,
        'File-Name': selectedFile.value.name
      }
    });

    if (response.ok) {
      const data = await response.json();
      if (response.status === 200) {
        scanStatus.value = '扫描成功';
        // 可以在这里处理扫描结果
      } else if (response.status === 202) {
        scanStatus.value = '文件扫描任务已接收，请稍后...';
        const md5Low32 = data.md5_low32;
        // 启动轮询
        pollInterval = setInterval(async () => {
          const pollResponse = await fetch(`api/api/check-report?md5_low32=${md5Low32}`, {
            method: 'GET',
            headers: {
              'Authorization': `${authStore.token}`
            }
          });

          if (pollResponse.ok) {
            const pollData = await pollResponse.json();
            if (pollResponse.status === 200) {
              scanStatus.value = '扫描成功';
              // 可以在这里处理扫描结果
              clearInterval(pollInterval as NodeJS.Timeout);
            }
          } else {
            // 报告尚未生成，继续轮询
          }
        }, 5000);
      }
    } else {
      scanStatus.value = `扫描失败，状态码: ${response.status}`;
    }
  } catch (error) {
    console.error('Failed to scan:', error);
  }
};

const downloadReport = async () => {
  if (!selectedFile.value) {
    scanStatus.value = '请先选择文件';
    return;
  }

  try {
    const response = await fetch('api/api/download-report', {
      method: 'GET',
      headers: {
        'Authorization': `${authStore.token}`,
        'File-Name': selectedFile.value.name
      }
    });

    if (response.ok) {
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = response.headers.get('File-Name')||"report.xlsx";
      // a.download = `report_${selectedFile.value.name}`;
      a.click();
      window.URL.revokeObjectURL(url);
      scanStatus.value = '报告下载成功';
    } else {
      scanStatus.value = `报告下载失败，状态码: ${response.status}`;
    }
  } catch (error) {
    console.error('Failed to download report:', error);
    scanStatus.value = '下载报告时出现错误';
  }
};

// 组件卸载时清除定时器
onUnmounted(() => {
  if (pollInterval) {
    clearInterval(pollInterval);
  }
});
</script>

<style scoped>
.file-upload-container {
  padding: 20px;
  background-color: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  max-width: 500px;
  margin: 20px auto;
}

.file-input {
  margin-bottom: 15px;
  display: block;
}

.btn {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  margin-right: 10px;
  transition: opacity 0.2s;
}

.btn-primary {
  background-color: #409EFF;
  color: white;
}

.btn-secondary {
  background-color: #67C23A;
  color: white;
}

.btn-success {
  background-color: #E6A23C;
  color: white;
}

.btn:hover {
  opacity: 0.9;
}

.status-message {
  margin-top: 10px;
  padding: 10px;
  border-radius: 4px;
}
</style>