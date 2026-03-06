export function Footer() {
  return (
    <footer className="fixed bottom-0 left-0 right-0 p-4 text-center z-40">
      <div className="inline-block backdrop-blur-md bg-white/20 dark:bg-black/20 border border-white/10 px-6 py-2 rounded-full text-sm text-gray-700 dark:text-gray-300 shadow-sm">
        © {new Date().getFullYear()} LinuxDoSpace (佬友空间) · 网站作者版权署名：墨叶·2026
      </div>
    </footer>
  );
}
