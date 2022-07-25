# idleconnv2
Idle conn v2

这个版本相对于第一代，完全放弃了用Deadline控制超时的做法，而改用自己计数，自己算超时。
一旦超时，采用终极中断措施，close。关掉链接。

其原理为：如果一个链接，建立之后，经过N秒，也没有任何读写动作，那就不需要占着茅坑不拉屎。我们关掉这样的僵死链接。


相反，如果我们定期有哪怕一个数据包读写动作，不管是read，还是write，只要在我们设定的时间阈值中间发生，那么我们自动续期一个周期。
直到下一个周期内，你再完成一次keepalive。
