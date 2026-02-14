export default function A2uiPage() {
  return (
    <div className="h-full overflow-auto p-6">
      <div className="mx-auto flex w-full max-w-4xl flex-col gap-6">
        <div className="rounded-xl border bg-white p-4">
          <div className="text-sm font-semibold text-gray-900">
            后端 A2UI 信息
          </div>
          <div className="mt-1 text-xs text-gray-500">
            统一管理 A2UI 定义、数据源与运行状态
          </div>
          <div className="mt-3 grid grid-cols-2 gap-3">
            <div className="rounded-md bg-gray-50 p-3">
              <div className="text-xs text-gray-500">已接入</div>
              <div className="text-lg font-semibold text-gray-900">0</div>
            </div>
            <div className="rounded-md bg-gray-50 p-3">
              <div className="text-xs text-gray-500">运行中</div>
              <div className="text-lg font-semibold text-gray-900">0</div>
            </div>
          </div>
        </div>
        <div className="rounded-xl border bg-white p-4">
          <div className="text-sm font-semibold text-gray-900">
            管理入口
          </div>
          <div className="mt-3 space-y-2 text-sm">
            <div className="flex items-center justify-between rounded-md border px-3 py-2 text-gray-700">
              <span>模型列表</span>
              <span className="text-xs text-gray-400">待接入</span>
            </div>
            <div className="flex items-center justify-between rounded-md border px-3 py-2 text-gray-700">
              <span>数据源管理</span>
              <span className="text-xs text-gray-400">待接入</span>
            </div>
            <div className="flex items-center justify-between rounded-md border px-3 py-2 text-gray-700">
              <span>运行监控</span>
              <span className="text-xs text-gray-400">待接入</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
