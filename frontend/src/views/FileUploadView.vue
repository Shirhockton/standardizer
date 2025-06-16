<template>
  <div class="file-upload-container">
    <input type="file" @change="handleFileChange" class="file-input" />
    <button @click="uploadFile" class="btn btn-primary">上传文件</button>
    <button @click="startScan" class="btn btn-secondary">开始扫描</button>
    <div v-if="uploadStatus" class="status-message">{{ uploadStatus }}</div>
    <div v-if="scanStatus" class="status-message">{{ scanStatus }}</div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useAuthStore } from '../store/auth';

const selectedFile = ref<File | null>(null);
const uploadStatus = ref('');
const scanStatus = ref('');

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
      scanStatus.value = '扫描成功';
    } else {
      scanStatus.value = `扫描失败，状态码: ${response.status}`;
    }
  } catch (error) {
    console.error('Failed to scan:', error);
  }
};
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

.btn:hover {
  opacity: 0.9;
}

.status-message {
  margin-top: 10px;
  padding: 10px;
  border-radius: 4px;
}
</style>