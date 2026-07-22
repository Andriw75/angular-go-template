import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { SidebarComponent } from '../sidebar/sidebar';
import { ToastComponent } from '../../../common/toast/toast';
import { ConfirmComponent } from '../../../common/confirm/confirm';

@Component({
  selector: 'app-dashboard-layout',
  templateUrl: './layout.html',
  styleUrl: './layout.css',
  imports: [RouterOutlet, SidebarComponent, ToastComponent, ConfirmComponent],
})
export class DashboardLayoutComponent {}
